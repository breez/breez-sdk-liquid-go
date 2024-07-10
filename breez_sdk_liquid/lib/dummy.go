// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-386"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-aarch"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-aarch64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/android-amd64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/darwin-aarch64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/darwin-amd64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/linux-aarch64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/linux-amd64"
	_ "github.com/breez/breez-sdk-liquid-go/breez_sdk_liquid/lib/windows-amd64"
)
