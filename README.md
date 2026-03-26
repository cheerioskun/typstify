[![license](https://img.shields.io/badge/license-Apache%20V2-green)](https://github.com/typstify/typstify/blob/main/LICENSE)

<p align="center"><img src="version/appicon.png" width="100" /></p>

# Typstify

The cross-platform desktop editor for Typst. Unlock the power of Typst with Typstify. Get the professional power of LaTeX with a modern, intuitive editor designed for seamless typesetting and development.


## Run

```sh
git clone https://github.com/typstify/typstify.git
cd typstify
go run .
```

To run the app locally, you must 
* Place the executables `typst` and `tinymist` (or `tyspt.exe` and `tinymist.exe`) in the root folder, 
* Or set custom executable paths for Typst and Tinymist in the setting page.


## Build

This project uses [Gio](https://gioui.org/) to build the UI. To build a binary release, you have to install and use the gogio tool, please 
refer to [gio-cmd](https://git.sr.ht/~eliasnaur/gio-cmd) to learn more. Also CGO must be enabled to build it.

**Important:** The typstify project is distributed as source code only. For pre-compiled binary releases, please download from the [official website](https://typstify.com/download)

## Contribute

Please feel free to contribute by filing issues or creating pull requests. 

## Explore Further

-	[Official Website](https://typstify.com)

## License

This project is distributed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0), see [LICENSE](./LICENSE) for more information.