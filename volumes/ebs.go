package volumes

import (
	"context"
	"errors"
	"log"
	"math"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/mvisonneau/go-ebsnvme/pkg/ebsnvme"
	"github.com/shirou/gopsutil/v3/disk"
)

type DiskData struct {
	VolumeID   string
	DeviceName string
	MountPoint string
	TotalUsed  float64
	TotalSpace uint64
	FsType     string
	VolumeSize int32
}

type Region struct {
	Region string `json:"region"`
}

var errVolumeRetryLater = errors.New("retry volume modification later")

func waitForEbsResize(volumeID string) error {
	client, err := getEc2Client()
	if err != nil {
		return err
	}
	input := &ec2.DescribeVolumesModificationsInput{VolumeIds: []string{volumeID}}
	status, err := client.DescribeVolumesModifications(context.Background(), input)
	if err != nil {
		return err
	}

	if status.VolumesModifications[0].ModificationState == "modifying" {
		log.Println("Ebs modification in progress. Waiting for 15 second")
		time.Sleep(15 * time.Second)
		if err := waitForEbsResize(volumeID); err != nil {
			return err
		}
	}
	return nil
}

func getEc2Client() (*ec2.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := imds.NewFromConfig(cfg)
	region, err := client.GetRegion(context.TODO(), &imds.GetRegionInput{})
	if err != nil {
		return nil, err
	}

	cfg.Region = region.Region
	return ec2.NewFromConfig(cfg), err
}

func getInstanceID() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	client := imds.NewFromConfig(cfg)
	instanceID, err := client.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return "", err
	}
	return instanceID.InstanceID, nil
}

func filterDisks() ([]DiskData, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	client, err := getEc2Client()
	if err != nil {
		return nil, err
	}

	instanceID, err := getInstanceID()
	if err != nil {
		return nil, err
	}
	diskData := []DiskData{}

	for _, p := range parts {
		var ebsDevice string

		if strings.Contains(p.Device, "nvme") {
			volumeMapping, err := ebsnvme.ScanDevice(p.Device)
			if err != nil {
				return nil, err
			}
			ebsDevice = volumeMapping.Name
		} else {
			ebsDevice = strings.Replace(p.Device, "xvd", "sd", 1)
		}

		filter := &ec2.DescribeVolumesInput{Filters: []types.Filter{
			{
				Name: aws.String("attachment.device"),
				Values: []string{
					ebsDevice,
				},
			},
			{
				Name: aws.String("attachment.instance-id"),
				Values: []string{
					instanceID,
				},
			},
		},
		}

		volumeInfo, err := client.DescribeVolumes(context.Background(), filter)
		if err != nil {
			return nil, err
		}
		usage, _ := disk.Usage(p.Mountpoint)
		disk := DiskData{
			VolumeID:   *volumeInfo.Volumes[0].VolumeId,
			DeviceName: p.Device,
			MountPoint: p.Mountpoint,
			TotalUsed:  usage.UsedPercent,
			TotalSpace: usage.Total,
			VolumeSize: volumeInfo.Volumes[0].Size,
			FsType:     p.Fstype,
		}
		diskData = append(diskData, disk)
	}
	return diskData, nil
}

func findNewSize(oldSize uint64, increasePercent float64) int32 {
	gbSize := float64(oldSize) / math.Pow(1024, 3)
	newSize := ((gbSize * increasePercent) / 100) + gbSize
	return int32(newSize)
}

func ebsResize(newSize int32, volumeID string) error {
	client, err := getEc2Client()
	if err != nil {
		return err
	}
	log.Println("Starting resize of ebs volume", volumeID)
	input := &ec2.ModifyVolumeInput{VolumeId: &volumeID, Size: newSize}
	if _, err := client.ModifyVolume(context.Background(), input); err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			switch ae.ErrorCode() {
			case "VolumeModificationRateExceeded":
				log.Println("Ebs was already resized, wait for 6 hours before next resize")
				return errVolumeRetryLater
			case "IncorrectModificationState":
				log.Println(ae.ErrorMessage())
				return errVolumeRetryLater
			}
		} else {
			return err
		}
	}
	err = waitForEbsResize(volumeID)
	if err != nil {
		return err
	}
	return nil
}

func fsResize(filesystem string, mountPoint string, partition string) error {
	log.Println("Starting system volume resize for", partition)
	var cmd *exec.Cmd
	if filesystem == "xfs" {
		cmd = exec.Command("xfs_growfs", "-d", mountPoint)
	} else {
		cmd = exec.Command("resize2fs", partition)
	}
	return cmd.Run()
}

func growPartition(partition string) error {
	log.Println("Starting growpart for", partition)
	var cmd *exec.Cmd
	i := strings.LastIndex(partition, "p")
	isLetter := regexp.MustCompile(`^/dev/+[a-zA-Z]+$`).MatchString
	if isLetter(partition) {
		log.Println("Grow partition for", partition, "not required")
	} else if i != -1 {
		cmd = exec.Command("growpart", partition[:i], partition[i+1:])
		return cmd.Run()
	} else if strings.Contains(partition, "xvd") {
		re := regexp.MustCompile(`\D+`)
		m := re.FindString(partition)
		cmd = exec.Command("growpart", m, partition[len(m):])
		return cmd.Run()
	}
	return nil
}

func ResizeDisk(increasePercent float64, freeSpaceThreshold float64) {
	disksInfo, err := filterDisks()
	if err != nil {
		log.Fatalln(err)
	}
	for _, disk := range disksInfo {
		if disk.TotalUsed < freeSpaceThreshold {
			log.Println("Resize for", disk.DeviceName, "not required")
			continue
		}

		log.Println("Starting resize of", disk.DeviceName)
		newSize := findNewSize(disk.TotalSpace, increasePercent)
		if err := ebsResize(int32(newSize), disk.VolumeID); err == errVolumeRetryLater {
			continue
		} else if err != nil {
			log.Fatalln(err)
		}
		if err := growPartition(disk.DeviceName); err != nil {
			log.Fatalln(err)
		}
		if err := fsResize(disk.FsType, disk.MountPoint, disk.DeviceName); err != nil {
			log.Fatalln(err)
		}
	}
}
