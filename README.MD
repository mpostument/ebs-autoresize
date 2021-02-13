# ebs-autoresize

Automatically resize your ebs.

## Table of Contents

- [ebs-autoresize](#ebs-autoresize)
  - [Table of Contents](#table-of-contents)
  - [Installing](#installing)
  - [Getting Started](#getting-started)
    - [Resize](#resize)
  - [Contributing](#contributing)
  - [License](#license)

## Installing

Download the latest binary from [releases](https://github.com/mpostument/ebs-autoresize/releases)

## Getting Started

### Resize

`ebs-autoresize --threshold=70 --increase=40`  
When amount of free space will be less then 70% disk will be resized on 40%

## Contributing

1. Fork it
2. Download your fork to your PC ( `git clone https://github.com/your_username/ebs-autoresize && cd ebs-autoresize` )
3. Create your feature branch ( `git checkout -b my-new-feature` )
4. Make changes and add them ( `git add .` )
5. Commit your changes ( `git commit -m 'Add some feature'` )
6. Push to the branch ( `git push origin my-new-feature` )
7. Create new pull request

## License

ebs-autoresize is released under the Apache 2.0 license. See [LICENSE.txt](https://github.com/mpostument/ebs-autoresize/blob/master/LICENSE)
