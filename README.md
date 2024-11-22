# Breez SDK - *Liquid*

## **Overview**

The Breez SDK provides developers with a end-to-end solution for integrating self-custodial Lightning payments into their apps and services. It eliminates the need for third-parties, simplifies the complexities of Bitcoin and Lightning, and enables seamless onboarding for billions of users to the future of peer-to-peer payments.

To provide the best experience for their end-users, developers can choose between the following implementations:

- [Breez SDK - *Liquid*](https://sdk-doc-liquid.breez.technology/)
- [Breez SDK - *Greenlight*](https://sdk-doc.breez.technology/)

**The Breez SDK is free for developers.**

## **What Is the *Liquid* Implementation?**

The *Liquid* implementation is a nodeless Lightning integration. It offers a self-custodial, end-to-end solution for integrating Lightning payments, utilizing the Liquid Network with on-chain interoperability and third-party fiat on-ramps.

**Core Functions**

- **Sending payments** *via protocols such as: bolt11, lnurl-pay, lightning address, btc address.*
- **Receiving payments** *via protocols such as: bolt11, lnurl-withdraw, btc address.*
- **Interacting with a wallet** *e.g. balance, max allow to pay, max allow to receive, on-chain balance.*

## Installation

To install the package:

```sh
$ go get github.com/breez/breez-sdk-liquid-go
```

### Supported platforms

This package embeds the Breez SDK - *Liquid* runtime compiled as shared library objects, and uses [`cgo`](https://golang.org/cmd/cgo/) to consume it. A set of precompiled shared library objects are provided. Thus this package works (and is tested) on the following platforms:

<table>
  <thead>
    <tr>
      <th>Platform</th>
      <th>Architecture</th>
      <th>Triple</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td rowspan="2">Android</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-linux-android</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-linux-android</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td rowspan="2">Darwin</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-apple-darwin</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-apple-darwin</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td rowspan="2">Linux</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-unknown-linux-gnu</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-unknown-linux-gnu</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td>Windows</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-pc-windows-msvc</code></td>
      <td>✅</td>
    </tr>
  </tbody>
</table>

## Usage

Head over to the [Breez SDK - Liquid documentation](https://sdk-doc-liquid.breez.technology/) to start implementing Lightning in your app.

```go
package main

import (
	"github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid"
)

func main() {
    mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

    config := breez_sdk_liquid.DefaultConfig(breez_sdk_liquid.LiquidNetworkTestnet)

    sdk, err := breez_sdk_liquid.Connect(breez_sdk_liquid.ConnectRequest{
        Config:   config,
        Mnemonic: mnemonic,
    })
}
```

## Bundling

For some platforms the provided binding libraries need to be copied into a location where they need to be found during runtime.

### Android

Copy the binding libraries into the jniLibs directory of your app
```bash
cp vendor/github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-386/*.so android/app/src/main/jniLibs/x86/
cp vendor/github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-aarch/*.so android/app/src/main/jniLibs/armeabi-v7a/
cp vendor/github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-aarch64/*.so android/app/src/main/jniLibs/arm64-v8a/
cp vendor/github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-amd64/*.so android/app/src/main/jniLibs/x86_64/
```
So they are in the following structure
```
└── android
    ├── app
        └── src
            └── main
                └── jniLibs
                    ├── arm64-v8a
                        ├── libbreez_sdk_liquid_bindings.so
                        └── libc++_shared.so
                    ├── armeabi-v7a
                        ├── libbreez_sdk_liquid_bindings.so
                        └── libc++_shared.so
                    ├── x86
                        ├── libbreez_sdk_liquid_bindings.so
                        └── libc++_shared.so
                    └── x86_64
                        ├── libbreez_sdk_liquid_bindings.so
                        └── libc++_shared.so
                └── AndroidManifest.xml
        └── build.gradle
    └── build.gradle
```

### Windows

Copy the binding library to the same directory as the executable file or include the library into the windows install packager.
```bash
cp vendor/github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/windows-amd64/*.dll build/windows/
```

## Information for Maintainers and Contributors

This repository is used to publish a Go package providing Go bindings to the Breez SDK - *Liquid*'s [underlying Rust implementation](https://github.com/breez/breez-sdk-liquid). The Go bindings are generated using [UniFFi Bindgen Go](https://github.com/NordSecurity/uniffi-bindgen-go).

Any changes to Breez SDK - *Liquid*, the Go bindings, and the configuration of this Go package must be made via the [breez-sdk-liquid](https://github.com/breez/breez-sdk-liquid) repository.
