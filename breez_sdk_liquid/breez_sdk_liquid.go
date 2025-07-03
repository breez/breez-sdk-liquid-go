package breez_sdk_liquid

// #include <breez_sdk_liquid.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// This is needed, because as of go 1.24
// type RustBuffer C.RustBuffer cannot have methods,
// RustBuffer is treated as non-local type
type GoRustBuffer struct {
	inner C.RustBuffer
}

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() uint64
	Capacity() uint64
}

func RustBufferFromExternal(b RustBufferI) GoRustBuffer {
	return GoRustBuffer{
		inner: C.RustBuffer{
			capacity: C.uint64_t(b.Capacity()),
			len:      C.uint64_t(b.Len()),
			data:     (*C.uchar)(b.Data()),
		},
	}
}

func (cb GoRustBuffer) Capacity() uint64 {
	return uint64(cb.inner.capacity)
}

func (cb GoRustBuffer) Len() uint64 {
	return uint64(cb.inner.len)
}

func (cb GoRustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.inner.data)
}

func (cb GoRustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.inner.data), C.uint64_t(cb.inner.len))
	return bytes.NewReader(b)
}

func (cb GoRustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_breez_sdk_liquid_bindings_rustbuffer_free(cb.inner, status)
		return false
	})
}

func (cb GoRustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.inner.data), C.int(cb.inner.len))
}

func stringToRustBuffer(str string) C.RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) C.RustBuffer {
	if len(b) == 0 {
		return C.RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) C.RustBuffer {
		return C.ffi_breez_sdk_liquid_bindings_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) C.RustBuffer
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) C.RustBuffer {
	// This might be not the most efficient way but it does not require knowing allocation size
	// beforehand
	var buffer bytes.Buffer
	bufWriter.Write(&buffer, value)

	bytes, err := io.ReadAll(&buffer)
	if err != nil {
		panic(fmt.Errorf("reading written data: %w", err))
	}
	return bytesToRustBuffer(bytes)
}

func LiftFromRustBuffer[GoType any](bufReader BufReader[GoType], rbuf RustBufferI) GoType {
	defer rbuf.Free()
	reader := rbuf.AsReader()
	item := bufReader.Read(reader)
	if reader.Len() > 0 {
		// TODO: Remove this
		leftover, _ := io.ReadAll(reader)
		panic(fmt.Errorf("Junk remaining in buffer after lifting: %s", string(leftover)))
	}
	return item
}

func rustCallWithError[E any, U any](converter BufReader[*E], callback func(*C.RustCallStatus) U) (U, *E) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)
	return returnValue, err
}

func checkCallStatus[E any](converter BufReader[*E], status C.RustCallStatus) *E {
	switch status.code {
	case 0:
		return nil
	case 1:
		return LiftFromRustBuffer(converter, GoRustBuffer{inner: status.errorBuf})
	case 2:
		// when the rust code sees a panic, it tries to construct a rustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{inner: status.errorBuf})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		panic(fmt.Errorf("unknown status code: %d", status.code))
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a C.RustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: status.errorBuf,
			})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError[error](nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
}

type NativeError interface {
	AsError() error
}

func writeInt8(writer io.Writer, value int8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint8(writer io.Writer, value uint8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt16(writer io.Writer, value int16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint16(writer io.Writer, value uint16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt32(writer io.Writer, value int32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint32(writer io.Writer, value uint32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt64(writer io.Writer, value int64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint64(writer io.Writer, value uint64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat32(writer io.Writer, value float32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat64(writer io.Writer, value float64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func readInt8(reader io.Reader) int8 {
	var result int8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint8(reader io.Reader) uint8 {
	var result uint8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt16(reader io.Reader) int16 {
	var result int16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint16(reader io.Reader) uint16 {
	var result uint16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt32(reader io.Reader) int32 {
	var result int32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint32(reader io.Reader) uint32 {
	var result uint32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt64(reader io.Reader) int64 {
	var result int64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint64(reader io.Reader) uint64 {
	var result uint64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat32(reader io.Reader) float32 {
	var result float32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat64(reader io.Reader) float64 {
	var result float64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func init() {

	FfiConverterCallbackInterfaceEventListenerINSTANCE.register()
	FfiConverterCallbackInterfaceLoggerINSTANCE.register()
	FfiConverterCallbackInterfaceSignerINSTANCE.register()
	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 26
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_breez_sdk_liquid_bindings_uniffi_contract_version()
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("breez_sdk_liquid: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_connect()
		})
		if checksum != 39960 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_connect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_connect_with_signer()
		})
		if checksum != 48633 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_connect_with_signer: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_default_config()
		})
		if checksum != 20931 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_default_config: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_parse_invoice()
		})
		if checksum != 45284 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_parse_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_set_logger()
		})
		if checksum != 32375 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_set_logger: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_accept_payment_proposed_fees()
		})
		if checksum != 57291 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_accept_payment_proposed_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_add_event_listener()
		})
		if checksum != 65289 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_add_event_listener: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_backup()
		})
		if checksum != 3592 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_buy_bitcoin()
		})
		if checksum != 53022 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_check_message()
		})
		if checksum != 64029 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_check_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_create_bolt12_invoice()
		})
		if checksum != 30488 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_create_bolt12_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_disconnect()
		})
		if checksum != 37717 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_disconnect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_fiat_rates()
		})
		if checksum != 61824 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_fiat_rates: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_lightning_limits()
		})
		if checksum != 61822 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_lightning_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_onchain_limits()
		})
		if checksum != 51575 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_onchain_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_payment_proposed_fees()
		})
		if checksum != 45806 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_payment_proposed_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_info()
		})
		if checksum != 4290 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_payment()
		})
		if checksum != 25832 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_fiat_currencies()
		})
		if checksum != 38203 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_fiat_currencies: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_payments()
		})
		if checksum != 39611 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_refundables()
		})
		if checksum != 22886 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_refundables: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_auth()
		})
		if checksum != 58655 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_auth: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_pay()
		})
		if checksum != 46650 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_pay: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_withdraw()
		})
		if checksum != 60533 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_withdraw: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_parse()
		})
		if checksum != 40166 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_parse: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_pay_onchain()
		})
		if checksum != 46079 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_buy_bitcoin()
		})
		if checksum != 26608 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_lnurl_pay()
		})
		if checksum != 14727 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_lnurl_pay: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_pay_onchain()
		})
		if checksum != 1876 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment()
		})
		if checksum != 28769 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_refund()
		})
		if checksum != 53467 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_send_payment()
		})
		if checksum != 1183 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_receive_payment()
		})
		if checksum != 63548 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_recommended_fees()
		})
		if checksum != 23255 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_recommended_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_refund()
		})
		if checksum != 31475 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_register_webhook()
		})
		if checksum != 3912 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_register_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_remove_event_listener()
		})
		if checksum != 16569 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_remove_event_listener: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_rescan_onchain_swaps()
		})
		if checksum != 14305 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_rescan_onchain_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_restore()
		})
		if checksum != 63590 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_restore: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_send_payment()
		})
		if checksum != 63087 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sign_message()
		})
		if checksum != 33731 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sign_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sync()
		})
		if checksum != 31783 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sync: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_unregister_webhook()
		})
		if checksum != 34970 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_unregister_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_eventlistener_on_event()
		})
		if checksum != 22441 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_eventlistener_on_event: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_logger_log()
		})
		if checksum != 36218 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_logger_log: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_xpub()
		})
		if checksum != 36847 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_xpub: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_derive_xpub()
		})
		if checksum != 8680 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_derive_xpub: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa()
		})
		if checksum != 48623 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa_recoverable()
		})
		if checksum != 263 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa_recoverable: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_slip77_master_blinding_key()
		})
		if checksum != 9707 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_slip77_master_blinding_key: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_hmac_sha256()
		})
		if checksum != 40934 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_hmac_sha256: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_encrypt()
		})
		if checksum != 43772 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_encrypt: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_decrypt()
		})
		if checksum != 45851 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_decrypt: UniFFI API checksum mismatch")
		}
	}
}

type FfiConverterUint8 struct{}

var FfiConverterUint8INSTANCE = FfiConverterUint8{}

func (FfiConverterUint8) Lower(value uint8) C.uint8_t {
	return C.uint8_t(value)
}

func (FfiConverterUint8) Write(writer io.Writer, value uint8) {
	writeUint8(writer, value)
}

func (FfiConverterUint8) Lift(value C.uint8_t) uint8 {
	return uint8(value)
}

func (FfiConverterUint8) Read(reader io.Reader) uint8 {
	return readUint8(reader)
}

type FfiDestroyerUint8 struct{}

func (FfiDestroyerUint8) Destroy(_ uint8) {}

type FfiConverterUint16 struct{}

var FfiConverterUint16INSTANCE = FfiConverterUint16{}

func (FfiConverterUint16) Lower(value uint16) C.uint16_t {
	return C.uint16_t(value)
}

func (FfiConverterUint16) Write(writer io.Writer, value uint16) {
	writeUint16(writer, value)
}

func (FfiConverterUint16) Lift(value C.uint16_t) uint16 {
	return uint16(value)
}

func (FfiConverterUint16) Read(reader io.Reader) uint16 {
	return readUint16(reader)
}

type FfiDestroyerUint16 struct{}

func (FfiDestroyerUint16) Destroy(_ uint16) {}

type FfiConverterUint32 struct{}

var FfiConverterUint32INSTANCE = FfiConverterUint32{}

func (FfiConverterUint32) Lower(value uint32) C.uint32_t {
	return C.uint32_t(value)
}

func (FfiConverterUint32) Write(writer io.Writer, value uint32) {
	writeUint32(writer, value)
}

func (FfiConverterUint32) Lift(value C.uint32_t) uint32 {
	return uint32(value)
}

func (FfiConverterUint32) Read(reader io.Reader) uint32 {
	return readUint32(reader)
}

type FfiDestroyerUint32 struct{}

func (FfiDestroyerUint32) Destroy(_ uint32) {}

type FfiConverterUint64 struct{}

var FfiConverterUint64INSTANCE = FfiConverterUint64{}

func (FfiConverterUint64) Lower(value uint64) C.uint64_t {
	return C.uint64_t(value)
}

func (FfiConverterUint64) Write(writer io.Writer, value uint64) {
	writeUint64(writer, value)
}

func (FfiConverterUint64) Lift(value C.uint64_t) uint64 {
	return uint64(value)
}

func (FfiConverterUint64) Read(reader io.Reader) uint64 {
	return readUint64(reader)
}

type FfiDestroyerUint64 struct{}

func (FfiDestroyerUint64) Destroy(_ uint64) {}

type FfiConverterInt64 struct{}

var FfiConverterInt64INSTANCE = FfiConverterInt64{}

func (FfiConverterInt64) Lower(value int64) C.int64_t {
	return C.int64_t(value)
}

func (FfiConverterInt64) Write(writer io.Writer, value int64) {
	writeInt64(writer, value)
}

func (FfiConverterInt64) Lift(value C.int64_t) int64 {
	return int64(value)
}

func (FfiConverterInt64) Read(reader io.Reader) int64 {
	return readInt64(reader)
}

type FfiDestroyerInt64 struct{}

func (FfiDestroyerInt64) Destroy(_ int64) {}

type FfiConverterFloat64 struct{}

var FfiConverterFloat64INSTANCE = FfiConverterFloat64{}

func (FfiConverterFloat64) Lower(value float64) C.double {
	return C.double(value)
}

func (FfiConverterFloat64) Write(writer io.Writer, value float64) {
	writeFloat64(writer, value)
}

func (FfiConverterFloat64) Lift(value C.double) float64 {
	return float64(value)
}

func (FfiConverterFloat64) Read(reader io.Reader) float64 {
	return readFloat64(reader)
}

type FfiDestroyerFloat64 struct{}

func (FfiDestroyerFloat64) Destroy(_ float64) {}

type FfiConverterBool struct{}

var FfiConverterBoolINSTANCE = FfiConverterBool{}

func (FfiConverterBool) Lower(value bool) C.int8_t {
	if value {
		return C.int8_t(1)
	}
	return C.int8_t(0)
}

func (FfiConverterBool) Write(writer io.Writer, value bool) {
	if value {
		writeInt8(writer, 1)
	} else {
		writeInt8(writer, 0)
	}
}

func (FfiConverterBool) Lift(value C.int8_t) bool {
	return value != 0
}

func (FfiConverterBool) Read(reader io.Reader) bool {
	return readInt8(reader) != 0
}

type FfiDestroyerBool struct{}

func (FfiDestroyerBool) Destroy(_ bool) {}

type FfiConverterString struct{}

var FfiConverterStringINSTANCE = FfiConverterString{}

func (FfiConverterString) Lift(rb RustBufferI) string {
	defer rb.Free()
	reader := rb.AsReader()
	b, err := io.ReadAll(reader)
	if err != nil {
		panic(fmt.Errorf("reading reader: %w", err))
	}
	return string(b)
}

func (FfiConverterString) Read(reader io.Reader) string {
	length := readInt32(reader)
	buffer := make([]byte, length)
	read_length, err := reader.Read(buffer)
	if err != nil {
		panic(err)
	}
	if read_length != int(length) {
		panic(fmt.Errorf("bad read length when reading string, expected %d, read %d", length, read_length))
	}
	return string(buffer)
}

func (FfiConverterString) Lower(value string) C.RustBuffer {
	return stringToRustBuffer(value)
}

func (FfiConverterString) Write(writer io.Writer, value string) {
	if len(value) > math.MaxInt32 {
		panic("String is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	write_length, err := io.WriteString(writer, value)
	if err != nil {
		panic(err)
	}
	if write_length != len(value) {
		panic(fmt.Errorf("bad write length when writing string, expected %d, written %d", len(value), write_length))
	}
}

type FfiDestroyerString struct{}

func (FfiDestroyerString) Destroy(_ string) {}

// Below is an implementation of synchronization requirements outlined in the link.
// https://github.com/mozilla/uniffi-rs/blob/0dc031132d9493ca812c3af6e7dd60ad2ea95bf0/uniffi_bindgen/src/bindings/kotlin/templates/ObjectRuntime.kt#L31

type FfiObject struct {
	pointer       unsafe.Pointer
	callCounter   atomic.Int64
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer
	freeFunction  func(unsafe.Pointer, *C.RustCallStatus)
	destroyed     atomic.Bool
}

func newFfiObject(
	pointer unsafe.Pointer,
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer,
	freeFunction func(unsafe.Pointer, *C.RustCallStatus),
) FfiObject {
	return FfiObject{
		pointer:       pointer,
		cloneFunction: cloneFunction,
		freeFunction:  freeFunction,
	}
}

func (ffiObject *FfiObject) incrementPointer(debugName string) unsafe.Pointer {
	for {
		counter := ffiObject.callCounter.Load()
		if counter <= -1 {
			panic(fmt.Errorf("%v object has already been destroyed", debugName))
		}
		if counter == math.MaxInt64 {
			panic(fmt.Errorf("%v object call counter would overflow", debugName))
		}
		if ffiObject.callCounter.CompareAndSwap(counter, counter+1) {
			break
		}
	}

	return rustCall(func(status *C.RustCallStatus) unsafe.Pointer {
		return ffiObject.cloneFunction(ffiObject.pointer, status)
	})
}

func (ffiObject *FfiObject) decrementPointer() {
	if ffiObject.callCounter.Add(-1) == -1 {
		ffiObject.freeRustArcPtr()
	}
}

func (ffiObject *FfiObject) destroy() {
	if ffiObject.destroyed.CompareAndSwap(false, true) {
		if ffiObject.callCounter.Add(-1) == -1 {
			ffiObject.freeRustArcPtr()
		}
	}
}

func (ffiObject *FfiObject) freeRustArcPtr() {
	rustCall(func(status *C.RustCallStatus) int32 {
		ffiObject.freeFunction(ffiObject.pointer, status)
		return 0
	})
}

type BindingLiquidSdkInterface interface {
	AcceptPaymentProposedFees(req AcceptPaymentProposedFeesRequest) *PaymentError
	AddEventListener(listener EventListener) (string, *SdkError)
	Backup(req BackupRequest) *SdkError
	BuyBitcoin(req BuyBitcoinRequest) (string, *PaymentError)
	CheckMessage(req CheckMessageRequest) (CheckMessageResponse, *SdkError)
	CreateBolt12Invoice(req CreateBolt12InvoiceRequest) (CreateBolt12InvoiceResponse, *PaymentError)
	Disconnect() *SdkError
	FetchFiatRates() ([]Rate, *SdkError)
	FetchLightningLimits() (LightningPaymentLimitsResponse, *PaymentError)
	FetchOnchainLimits() (OnchainPaymentLimitsResponse, *PaymentError)
	FetchPaymentProposedFees(req FetchPaymentProposedFeesRequest) (FetchPaymentProposedFeesResponse, *SdkError)
	GetInfo() (GetInfoResponse, *SdkError)
	GetPayment(req GetPaymentRequest) (*Payment, *PaymentError)
	ListFiatCurrencies() ([]FiatCurrency, *SdkError)
	ListPayments(req ListPaymentsRequest) ([]Payment, *PaymentError)
	ListRefundables() ([]RefundableSwap, *SdkError)
	LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, *LnUrlAuthError)
	LnurlPay(req LnUrlPayRequest) (LnUrlPayResult, *LnUrlPayError)
	LnurlWithdraw(req LnUrlWithdrawRequest) (LnUrlWithdrawResult, *LnUrlWithdrawError)
	Parse(input string) (InputType, *PaymentError)
	PayOnchain(req PayOnchainRequest) (SendPaymentResponse, *PaymentError)
	PrepareBuyBitcoin(req PrepareBuyBitcoinRequest) (PrepareBuyBitcoinResponse, *PaymentError)
	PrepareLnurlPay(req PrepareLnUrlPayRequest) (PrepareLnUrlPayResponse, *LnUrlPayError)
	PreparePayOnchain(req PreparePayOnchainRequest) (PreparePayOnchainResponse, *PaymentError)
	PrepareReceivePayment(req PrepareReceiveRequest) (PrepareReceiveResponse, *PaymentError)
	PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, *SdkError)
	PrepareSendPayment(req PrepareSendRequest) (PrepareSendResponse, *PaymentError)
	ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, *PaymentError)
	RecommendedFees() (RecommendedFees, *SdkError)
	Refund(req RefundRequest) (RefundResponse, *PaymentError)
	RegisterWebhook(webhookUrl string) *SdkError
	RemoveEventListener(id string) *SdkError
	RescanOnchainSwaps() *SdkError
	Restore(req RestoreRequest) *SdkError
	SendPayment(req SendPaymentRequest) (SendPaymentResponse, *PaymentError)
	SignMessage(req SignMessageRequest) (SignMessageResponse, *SdkError)
	Sync() *SdkError
	UnregisterWebhook() *SdkError
}
type BindingLiquidSdk struct {
	ffiObject FfiObject
}

func (_self *BindingLiquidSdk) AcceptPaymentProposedFees(req AcceptPaymentProposedFeesRequest) *PaymentError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_accept_payment_proposed_fees(
			_pointer, FfiConverterAcceptPaymentProposedFeesRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) AddEventListener(listener EventListener) (string, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_add_event_listener(
				_pointer, FfiConverterCallbackInterfaceEventListenerINSTANCE.Lower(listener), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Backup(req BackupRequest) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_backup(
			_pointer, FfiConverterBackupRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) BuyBitcoin(req BuyBitcoinRequest) (string, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_buy_bitcoin(
				_pointer, FfiConverterBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) CheckMessage(req CheckMessageRequest) (CheckMessageResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_check_message(
				_pointer, FfiConverterCheckMessageRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue CheckMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterCheckMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) CreateBolt12Invoice(req CreateBolt12InvoiceRequest) (CreateBolt12InvoiceResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_create_bolt12_invoice(
				_pointer, FfiConverterCreateBolt12InvoiceRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue CreateBolt12InvoiceResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterCreateBolt12InvoiceResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Disconnect() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_disconnect(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) FetchFiatRates() ([]Rate, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_fiat_rates(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Rate
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceRateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchLightningLimits() (LightningPaymentLimitsResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_lightning_limits(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LightningPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLightningPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchOnchainLimits() (OnchainPaymentLimitsResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_onchain_limits(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OnchainPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOnchainPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchPaymentProposedFees(req FetchPaymentProposedFeesRequest) (FetchPaymentProposedFeesResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_payment_proposed_fees(
				_pointer, FfiConverterFetchPaymentProposedFeesRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FetchPaymentProposedFeesResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterFetchPaymentProposedFeesResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) GetInfo() (GetInfoResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_get_info(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue GetInfoResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterGetInfoResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) GetPayment(req GetPaymentRequest) (*Payment, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_get_payment(
				_pointer, FfiConverterGetPaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalPaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListFiatCurrencies() ([]FiatCurrency, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_fiat_currencies(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []FiatCurrency
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceFiatCurrencyINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListPayments(req ListPaymentsRequest) ([]Payment, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_payments(
				_pointer, FfiConverterListPaymentsRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequencePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListRefundables() ([]RefundableSwap, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_refundables(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []RefundableSwap
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceRefundableSwapINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, *LnUrlAuthError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlAuthError](FfiConverterLnUrlAuthError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_auth(
				_pointer, FfiConverterLnUrlAuthRequestDataINSTANCE.Lower(reqData), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlCallbackStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlCallbackStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlPay(req LnUrlPayRequest) (LnUrlPayResult, *LnUrlPayError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlPayError](FfiConverterLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_pay(
				_pointer, FfiConverterLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlPayResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlPayResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlWithdraw(req LnUrlWithdrawRequest) (LnUrlWithdrawResult, *LnUrlWithdrawError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlWithdrawError](FfiConverterLnUrlWithdrawError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_withdraw(
				_pointer, FfiConverterLnUrlWithdrawRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlWithdrawResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlWithdrawResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Parse(input string) (InputType, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_parse(
				_pointer, FfiConverterStringINSTANCE.Lower(input), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue InputType
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterInputTypeINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PayOnchain(req PayOnchainRequest) (SendPaymentResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_pay_onchain(
				_pointer, FfiConverterPayOnchainRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareBuyBitcoin(req PrepareBuyBitcoinRequest) (PrepareBuyBitcoinResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_buy_bitcoin(
				_pointer, FfiConverterPrepareBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareBuyBitcoinResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareBuyBitcoinResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareLnurlPay(req PrepareLnUrlPayRequest) (PrepareLnUrlPayResponse, *LnUrlPayError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlPayError](FfiConverterLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_lnurl_pay(
				_pointer, FfiConverterPrepareLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareLnUrlPayResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareLnUrlPayResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PreparePayOnchain(req PreparePayOnchainRequest) (PreparePayOnchainResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_pay_onchain(
				_pointer, FfiConverterPreparePayOnchainRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PreparePayOnchainResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPreparePayOnchainResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareReceivePayment(req PrepareReceiveRequest) (PrepareReceiveResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_receive_payment(
				_pointer, FfiConverterPrepareReceiveRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareReceiveResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareReceiveResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_refund(
				_pointer, FfiConverterPrepareRefundRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareSendPayment(req PrepareSendRequest) (PrepareSendResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_send_payment(
				_pointer, FfiConverterPrepareSendRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareSendResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareSendResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_receive_payment(
				_pointer, FfiConverterReceivePaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReceivePaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterReceivePaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) RecommendedFees() (RecommendedFees, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_recommended_fees(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RecommendedFees
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterRecommendedFeesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Refund(req RefundRequest) (RefundResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_refund(
				_pointer, FfiConverterRefundRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) RegisterWebhook(webhookUrl string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_register_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) RemoveEventListener(id string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_remove_event_listener(
			_pointer, FfiConverterStringINSTANCE.Lower(id), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) RescanOnchainSwaps() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_rescan_onchain_swaps(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) Restore(req RestoreRequest) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_restore(
			_pointer, FfiConverterRestoreRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) SendPayment(req SendPaymentRequest) (SendPaymentResponse, *PaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_send_payment(
				_pointer, FfiConverterSendPaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) SignMessage(req SignMessageRequest) (SignMessageResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_sign_message(
				_pointer, FfiConverterSignMessageRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SignMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSignMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Sync() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_sync(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) UnregisterWebhook() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_unregister_webhook(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}
func (object *BindingLiquidSdk) Destroy() {
	runtime.SetFinalizer(object, nil)
	object.ffiObject.destroy()
}

type FfiConverterBindingLiquidSdk struct{}

var FfiConverterBindingLiquidSdkINSTANCE = FfiConverterBindingLiquidSdk{}

func (c FfiConverterBindingLiquidSdk) Lift(pointer unsafe.Pointer) *BindingLiquidSdk {
	result := &BindingLiquidSdk{
		newFfiObject(
			pointer,
			func(pointer unsafe.Pointer, status *C.RustCallStatus) unsafe.Pointer {
				return C.uniffi_breez_sdk_liquid_bindings_fn_clone_bindingliquidsdk(pointer, status)
			},
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_breez_sdk_liquid_bindings_fn_free_bindingliquidsdk(pointer, status)
			},
		),
	}
	runtime.SetFinalizer(result, (*BindingLiquidSdk).Destroy)
	return result
}

func (c FfiConverterBindingLiquidSdk) Read(reader io.Reader) *BindingLiquidSdk {
	return c.Lift(unsafe.Pointer(uintptr(readUint64(reader))))
}

func (c FfiConverterBindingLiquidSdk) Lower(value *BindingLiquidSdk) unsafe.Pointer {
	// TODO: this is bad - all synchronization from ObjectRuntime.go is discarded here,
	// because the pointer will be decremented immediately after this function returns,
	// and someone will be left holding onto a non-locked pointer.
	pointer := value.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer value.ffiObject.decrementPointer()
	return pointer

}

func (c FfiConverterBindingLiquidSdk) Write(writer io.Writer, value *BindingLiquidSdk) {
	writeUint64(writer, uint64(uintptr(c.Lower(value))))
}

type FfiDestroyerBindingLiquidSdk struct{}

func (_ FfiDestroyerBindingLiquidSdk) Destroy(value *BindingLiquidSdk) {
	value.Destroy()
}

type AcceptPaymentProposedFeesRequest struct {
	Response FetchPaymentProposedFeesResponse
}

func (r *AcceptPaymentProposedFeesRequest) Destroy() {
	FfiDestroyerFetchPaymentProposedFeesResponse{}.Destroy(r.Response)
}

type FfiConverterAcceptPaymentProposedFeesRequest struct{}

var FfiConverterAcceptPaymentProposedFeesRequestINSTANCE = FfiConverterAcceptPaymentProposedFeesRequest{}

func (c FfiConverterAcceptPaymentProposedFeesRequest) Lift(rb RustBufferI) AcceptPaymentProposedFeesRequest {
	return LiftFromRustBuffer[AcceptPaymentProposedFeesRequest](c, rb)
}

func (c FfiConverterAcceptPaymentProposedFeesRequest) Read(reader io.Reader) AcceptPaymentProposedFeesRequest {
	return AcceptPaymentProposedFeesRequest{
		FfiConverterFetchPaymentProposedFeesResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterAcceptPaymentProposedFeesRequest) Lower(value AcceptPaymentProposedFeesRequest) C.RustBuffer {
	return LowerIntoRustBuffer[AcceptPaymentProposedFeesRequest](c, value)
}

func (c FfiConverterAcceptPaymentProposedFeesRequest) Write(writer io.Writer, value AcceptPaymentProposedFeesRequest) {
	FfiConverterFetchPaymentProposedFeesResponseINSTANCE.Write(writer, value.Response)
}

type FfiDestroyerAcceptPaymentProposedFeesRequest struct{}

func (_ FfiDestroyerAcceptPaymentProposedFeesRequest) Destroy(value AcceptPaymentProposedFeesRequest) {
	value.Destroy()
}

type AesSuccessActionData struct {
	Description string
	Ciphertext  string
	Iv          string
}

func (r *AesSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Ciphertext)
	FfiDestroyerString{}.Destroy(r.Iv)
}

type FfiConverterAesSuccessActionData struct{}

var FfiConverterAesSuccessActionDataINSTANCE = FfiConverterAesSuccessActionData{}

func (c FfiConverterAesSuccessActionData) Lift(rb RustBufferI) AesSuccessActionData {
	return LiftFromRustBuffer[AesSuccessActionData](c, rb)
}

func (c FfiConverterAesSuccessActionData) Read(reader io.Reader) AesSuccessActionData {
	return AesSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterAesSuccessActionData) Lower(value AesSuccessActionData) C.RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionData](c, value)
}

func (c FfiConverterAesSuccessActionData) Write(writer io.Writer, value AesSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Ciphertext)
	FfiConverterStringINSTANCE.Write(writer, value.Iv)
}

type FfiDestroyerAesSuccessActionData struct{}

func (_ FfiDestroyerAesSuccessActionData) Destroy(value AesSuccessActionData) {
	value.Destroy()
}

type AesSuccessActionDataDecrypted struct {
	Description string
	Plaintext   string
}

func (r *AesSuccessActionDataDecrypted) Destroy() {
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Plaintext)
}

type FfiConverterAesSuccessActionDataDecrypted struct{}

var FfiConverterAesSuccessActionDataDecryptedINSTANCE = FfiConverterAesSuccessActionDataDecrypted{}

func (c FfiConverterAesSuccessActionDataDecrypted) Lift(rb RustBufferI) AesSuccessActionDataDecrypted {
	return LiftFromRustBuffer[AesSuccessActionDataDecrypted](c, rb)
}

func (c FfiConverterAesSuccessActionDataDecrypted) Read(reader io.Reader) AesSuccessActionDataDecrypted {
	return AesSuccessActionDataDecrypted{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterAesSuccessActionDataDecrypted) Lower(value AesSuccessActionDataDecrypted) C.RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataDecrypted](c, value)
}

func (c FfiConverterAesSuccessActionDataDecrypted) Write(writer io.Writer, value AesSuccessActionDataDecrypted) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Plaintext)
}

type FfiDestroyerAesSuccessActionDataDecrypted struct{}

func (_ FfiDestroyerAesSuccessActionDataDecrypted) Destroy(value AesSuccessActionDataDecrypted) {
	value.Destroy()
}

type AssetBalance struct {
	AssetId    string
	BalanceSat uint64
	Name       *string
	Ticker     *string
	Balance    *float64
}

func (r *AssetBalance) Destroy() {
	FfiDestroyerString{}.Destroy(r.AssetId)
	FfiDestroyerUint64{}.Destroy(r.BalanceSat)
	FfiDestroyerOptionalString{}.Destroy(r.Name)
	FfiDestroyerOptionalString{}.Destroy(r.Ticker)
	FfiDestroyerOptionalFloat64{}.Destroy(r.Balance)
}

type FfiConverterAssetBalance struct{}

var FfiConverterAssetBalanceINSTANCE = FfiConverterAssetBalance{}

func (c FfiConverterAssetBalance) Lift(rb RustBufferI) AssetBalance {
	return LiftFromRustBuffer[AssetBalance](c, rb)
}

func (c FfiConverterAssetBalance) Read(reader io.Reader) AssetBalance {
	return AssetBalance{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterAssetBalance) Lower(value AssetBalance) C.RustBuffer {
	return LowerIntoRustBuffer[AssetBalance](c, value)
}

func (c FfiConverterAssetBalance) Write(writer io.Writer, value AssetBalance) {
	FfiConverterStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterUint64INSTANCE.Write(writer, value.BalanceSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Name)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.Balance)
}

type FfiDestroyerAssetBalance struct{}

func (_ FfiDestroyerAssetBalance) Destroy(value AssetBalance) {
	value.Destroy()
}

type AssetInfo struct {
	Name   string
	Ticker string
	Amount float64
	Fees   *float64
}

func (r *AssetInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerString{}.Destroy(r.Ticker)
	FfiDestroyerFloat64{}.Destroy(r.Amount)
	FfiDestroyerOptionalFloat64{}.Destroy(r.Fees)
}

type FfiConverterAssetInfo struct{}

var FfiConverterAssetInfoINSTANCE = FfiConverterAssetInfo{}

func (c FfiConverterAssetInfo) Lift(rb RustBufferI) AssetInfo {
	return LiftFromRustBuffer[AssetInfo](c, rb)
}

func (c FfiConverterAssetInfo) Read(reader io.Reader) AssetInfo {
	return AssetInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterAssetInfo) Lower(value AssetInfo) C.RustBuffer {
	return LowerIntoRustBuffer[AssetInfo](c, value)
}

func (c FfiConverterAssetInfo) Write(writer io.Writer, value AssetInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterFloat64INSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.Fees)
}

type FfiDestroyerAssetInfo struct{}

func (_ FfiDestroyerAssetInfo) Destroy(value AssetInfo) {
	value.Destroy()
}

type AssetMetadata struct {
	AssetId   string
	Name      string
	Ticker    string
	Precision uint8
	FiatId    *string
}

func (r *AssetMetadata) Destroy() {
	FfiDestroyerString{}.Destroy(r.AssetId)
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerString{}.Destroy(r.Ticker)
	FfiDestroyerUint8{}.Destroy(r.Precision)
	FfiDestroyerOptionalString{}.Destroy(r.FiatId)
}

type FfiConverterAssetMetadata struct{}

var FfiConverterAssetMetadataINSTANCE = FfiConverterAssetMetadata{}

func (c FfiConverterAssetMetadata) Lift(rb RustBufferI) AssetMetadata {
	return LiftFromRustBuffer[AssetMetadata](c, rb)
}

func (c FfiConverterAssetMetadata) Read(reader io.Reader) AssetMetadata {
	return AssetMetadata{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint8INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterAssetMetadata) Lower(value AssetMetadata) C.RustBuffer {
	return LowerIntoRustBuffer[AssetMetadata](c, value)
}

func (c FfiConverterAssetMetadata) Write(writer io.Writer, value AssetMetadata) {
	FfiConverterStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterUint8INSTANCE.Write(writer, value.Precision)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.FiatId)
}

type FfiDestroyerAssetMetadata struct{}

func (_ FfiDestroyerAssetMetadata) Destroy(value AssetMetadata) {
	value.Destroy()
}

type BackupRequest struct {
	BackupPath *string
}

func (r *BackupRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.BackupPath)
}

type FfiConverterBackupRequest struct{}

var FfiConverterBackupRequestINSTANCE = FfiConverterBackupRequest{}

func (c FfiConverterBackupRequest) Lift(rb RustBufferI) BackupRequest {
	return LiftFromRustBuffer[BackupRequest](c, rb)
}

func (c FfiConverterBackupRequest) Read(reader io.Reader) BackupRequest {
	return BackupRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBackupRequest) Lower(value BackupRequest) C.RustBuffer {
	return LowerIntoRustBuffer[BackupRequest](c, value)
}

func (c FfiConverterBackupRequest) Write(writer io.Writer, value BackupRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BackupPath)
}

type FfiDestroyerBackupRequest struct{}

func (_ FfiDestroyerBackupRequest) Destroy(value BackupRequest) {
	value.Destroy()
}

type BitcoinAddressData struct {
	Address   string
	Network   Network
	AmountSat *uint64
	Label     *string
	Message   *string
}

func (r *BitcoinAddressData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterBitcoinAddressData struct{}

var FfiConverterBitcoinAddressDataINSTANCE = FfiConverterBitcoinAddressData{}

func (c FfiConverterBitcoinAddressData) Lift(rb RustBufferI) BitcoinAddressData {
	return LiftFromRustBuffer[BitcoinAddressData](c, rb)
}

func (c FfiConverterBitcoinAddressData) Read(reader io.Reader) BitcoinAddressData {
	return BitcoinAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBitcoinAddressData) Lower(value BitcoinAddressData) C.RustBuffer {
	return LowerIntoRustBuffer[BitcoinAddressData](c, value)
}

func (c FfiConverterBitcoinAddressData) Write(writer io.Writer, value BitcoinAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerBitcoinAddressData struct{}

func (_ FfiDestroyerBitcoinAddressData) Destroy(value BitcoinAddressData) {
	value.Destroy()
}

type BlockchainInfo struct {
	LiquidTip  uint32
	BitcoinTip uint32
}

func (r *BlockchainInfo) Destroy() {
	FfiDestroyerUint32{}.Destroy(r.LiquidTip)
	FfiDestroyerUint32{}.Destroy(r.BitcoinTip)
}

type FfiConverterBlockchainInfo struct{}

var FfiConverterBlockchainInfoINSTANCE = FfiConverterBlockchainInfo{}

func (c FfiConverterBlockchainInfo) Lift(rb RustBufferI) BlockchainInfo {
	return LiftFromRustBuffer[BlockchainInfo](c, rb)
}

func (c FfiConverterBlockchainInfo) Read(reader io.Reader) BlockchainInfo {
	return BlockchainInfo{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterBlockchainInfo) Lower(value BlockchainInfo) C.RustBuffer {
	return LowerIntoRustBuffer[BlockchainInfo](c, value)
}

func (c FfiConverterBlockchainInfo) Write(writer io.Writer, value BlockchainInfo) {
	FfiConverterUint32INSTANCE.Write(writer, value.LiquidTip)
	FfiConverterUint32INSTANCE.Write(writer, value.BitcoinTip)
}

type FfiDestroyerBlockchainInfo struct{}

func (_ FfiDestroyerBlockchainInfo) Destroy(value BlockchainInfo) {
	value.Destroy()
}

type BuyBitcoinRequest struct {
	PrepareResponse PrepareBuyBitcoinResponse
	RedirectUrl     *string
}

func (r *BuyBitcoinRequest) Destroy() {
	FfiDestroyerPrepareBuyBitcoinResponse{}.Destroy(r.PrepareResponse)
	FfiDestroyerOptionalString{}.Destroy(r.RedirectUrl)
}

type FfiConverterBuyBitcoinRequest struct{}

var FfiConverterBuyBitcoinRequestINSTANCE = FfiConverterBuyBitcoinRequest{}

func (c FfiConverterBuyBitcoinRequest) Lift(rb RustBufferI) BuyBitcoinRequest {
	return LiftFromRustBuffer[BuyBitcoinRequest](c, rb)
}

func (c FfiConverterBuyBitcoinRequest) Read(reader io.Reader) BuyBitcoinRequest {
	return BuyBitcoinRequest{
		FfiConverterPrepareBuyBitcoinResponseINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBuyBitcoinRequest) Lower(value BuyBitcoinRequest) C.RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinRequest](c, value)
}

func (c FfiConverterBuyBitcoinRequest) Write(writer io.Writer, value BuyBitcoinRequest) {
	FfiConverterPrepareBuyBitcoinResponseINSTANCE.Write(writer, value.PrepareResponse)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.RedirectUrl)
}

type FfiDestroyerBuyBitcoinRequest struct{}

func (_ FfiDestroyerBuyBitcoinRequest) Destroy(value BuyBitcoinRequest) {
	value.Destroy()
}

type CheckMessageRequest struct {
	Message   string
	Pubkey    string
	Signature string
}

func (r *CheckMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
	FfiDestroyerString{}.Destroy(r.Pubkey)
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterCheckMessageRequest struct{}

var FfiConverterCheckMessageRequestINSTANCE = FfiConverterCheckMessageRequest{}

func (c FfiConverterCheckMessageRequest) Lift(rb RustBufferI) CheckMessageRequest {
	return LiftFromRustBuffer[CheckMessageRequest](c, rb)
}

func (c FfiConverterCheckMessageRequest) Read(reader io.Reader) CheckMessageRequest {
	return CheckMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterCheckMessageRequest) Lower(value CheckMessageRequest) C.RustBuffer {
	return LowerIntoRustBuffer[CheckMessageRequest](c, value)
}

func (c FfiConverterCheckMessageRequest) Write(writer io.Writer, value CheckMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerCheckMessageRequest struct{}

func (_ FfiDestroyerCheckMessageRequest) Destroy(value CheckMessageRequest) {
	value.Destroy()
}

type CheckMessageResponse struct {
	IsValid bool
}

func (r *CheckMessageResponse) Destroy() {
	FfiDestroyerBool{}.Destroy(r.IsValid)
}

type FfiConverterCheckMessageResponse struct{}

var FfiConverterCheckMessageResponseINSTANCE = FfiConverterCheckMessageResponse{}

func (c FfiConverterCheckMessageResponse) Lift(rb RustBufferI) CheckMessageResponse {
	return LiftFromRustBuffer[CheckMessageResponse](c, rb)
}

func (c FfiConverterCheckMessageResponse) Read(reader io.Reader) CheckMessageResponse {
	return CheckMessageResponse{
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterCheckMessageResponse) Lower(value CheckMessageResponse) C.RustBuffer {
	return LowerIntoRustBuffer[CheckMessageResponse](c, value)
}

func (c FfiConverterCheckMessageResponse) Write(writer io.Writer, value CheckMessageResponse) {
	FfiConverterBoolINSTANCE.Write(writer, value.IsValid)
}

type FfiDestroyerCheckMessageResponse struct{}

func (_ FfiDestroyerCheckMessageResponse) Destroy(value CheckMessageResponse) {
	value.Destroy()
}

type Config struct {
	LiquidExplorer                 BlockchainExplorer
	BitcoinExplorer                BlockchainExplorer
	WorkingDir                     string
	Network                        LiquidNetwork
	PaymentTimeoutSec              uint64
	SyncServiceUrl                 *string
	BreezApiKey                    *string
	ZeroConfMaxAmountSat           *uint64
	UseDefaultExternalInputParsers bool
	ExternalInputParsers           *[]ExternalInputParser
	OnchainFeeRateLeewaySat        *uint64
	AssetMetadata                  *[]AssetMetadata
	SideswapApiKey                 *string
}

func (r *Config) Destroy() {
	FfiDestroyerBlockchainExplorer{}.Destroy(r.LiquidExplorer)
	FfiDestroyerBlockchainExplorer{}.Destroy(r.BitcoinExplorer)
	FfiDestroyerString{}.Destroy(r.WorkingDir)
	FfiDestroyerLiquidNetwork{}.Destroy(r.Network)
	FfiDestroyerUint64{}.Destroy(r.PaymentTimeoutSec)
	FfiDestroyerOptionalString{}.Destroy(r.SyncServiceUrl)
	FfiDestroyerOptionalString{}.Destroy(r.BreezApiKey)
	FfiDestroyerOptionalUint64{}.Destroy(r.ZeroConfMaxAmountSat)
	FfiDestroyerBool{}.Destroy(r.UseDefaultExternalInputParsers)
	FfiDestroyerOptionalSequenceExternalInputParser{}.Destroy(r.ExternalInputParsers)
	FfiDestroyerOptionalUint64{}.Destroy(r.OnchainFeeRateLeewaySat)
	FfiDestroyerOptionalSequenceAssetMetadata{}.Destroy(r.AssetMetadata)
	FfiDestroyerOptionalString{}.Destroy(r.SideswapApiKey)
}

type FfiConverterConfig struct{}

var FfiConverterConfigINSTANCE = FfiConverterConfig{}

func (c FfiConverterConfig) Lift(rb RustBufferI) Config {
	return LiftFromRustBuffer[Config](c, rb)
}

func (c FfiConverterConfig) Read(reader io.Reader) Config {
	return Config{
		FfiConverterBlockchainExplorerINSTANCE.Read(reader),
		FfiConverterBlockchainExplorerINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterLiquidNetworkINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalSequenceExternalInputParserINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalSequenceAssetMetadataINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterConfig) Lower(value Config) C.RustBuffer {
	return LowerIntoRustBuffer[Config](c, value)
}

func (c FfiConverterConfig) Write(writer io.Writer, value Config) {
	FfiConverterBlockchainExplorerINSTANCE.Write(writer, value.LiquidExplorer)
	FfiConverterBlockchainExplorerINSTANCE.Write(writer, value.BitcoinExplorer)
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
	FfiConverterLiquidNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterUint64INSTANCE.Write(writer, value.PaymentTimeoutSec)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.SyncServiceUrl)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BreezApiKey)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.ZeroConfMaxAmountSat)
	FfiConverterBoolINSTANCE.Write(writer, value.UseDefaultExternalInputParsers)
	FfiConverterOptionalSequenceExternalInputParserINSTANCE.Write(writer, value.ExternalInputParsers)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.OnchainFeeRateLeewaySat)
	FfiConverterOptionalSequenceAssetMetadataINSTANCE.Write(writer, value.AssetMetadata)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.SideswapApiKey)
}

type FfiDestroyerConfig struct{}

func (_ FfiDestroyerConfig) Destroy(value Config) {
	value.Destroy()
}

type ConnectRequest struct {
	Config     Config
	Mnemonic   *string
	Passphrase *string
	Seed       *[]uint8
}

func (r *ConnectRequest) Destroy() {
	FfiDestroyerConfig{}.Destroy(r.Config)
	FfiDestroyerOptionalString{}.Destroy(r.Mnemonic)
	FfiDestroyerOptionalString{}.Destroy(r.Passphrase)
	FfiDestroyerOptionalSequenceUint8{}.Destroy(r.Seed)
}

type FfiConverterConnectRequest struct{}

var FfiConverterConnectRequestINSTANCE = FfiConverterConnectRequest{}

func (c FfiConverterConnectRequest) Lift(rb RustBufferI) ConnectRequest {
	return LiftFromRustBuffer[ConnectRequest](c, rb)
}

func (c FfiConverterConnectRequest) Read(reader io.Reader) ConnectRequest {
	return ConnectRequest{
		FfiConverterConfigINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterConnectRequest) Lower(value ConnectRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ConnectRequest](c, value)
}

func (c FfiConverterConnectRequest) Write(writer io.Writer, value ConnectRequest) {
	FfiConverterConfigINSTANCE.Write(writer, value.Config)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Mnemonic)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Passphrase)
	FfiConverterOptionalSequenceUint8INSTANCE.Write(writer, value.Seed)
}

type FfiDestroyerConnectRequest struct{}

func (_ FfiDestroyerConnectRequest) Destroy(value ConnectRequest) {
	value.Destroy()
}

type ConnectWithSignerRequest struct {
	Config Config
}

func (r *ConnectWithSignerRequest) Destroy() {
	FfiDestroyerConfig{}.Destroy(r.Config)
}

type FfiConverterConnectWithSignerRequest struct{}

var FfiConverterConnectWithSignerRequestINSTANCE = FfiConverterConnectWithSignerRequest{}

func (c FfiConverterConnectWithSignerRequest) Lift(rb RustBufferI) ConnectWithSignerRequest {
	return LiftFromRustBuffer[ConnectWithSignerRequest](c, rb)
}

func (c FfiConverterConnectWithSignerRequest) Read(reader io.Reader) ConnectWithSignerRequest {
	return ConnectWithSignerRequest{
		FfiConverterConfigINSTANCE.Read(reader),
	}
}

func (c FfiConverterConnectWithSignerRequest) Lower(value ConnectWithSignerRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ConnectWithSignerRequest](c, value)
}

func (c FfiConverterConnectWithSignerRequest) Write(writer io.Writer, value ConnectWithSignerRequest) {
	FfiConverterConfigINSTANCE.Write(writer, value.Config)
}

type FfiDestroyerConnectWithSignerRequest struct{}

func (_ FfiDestroyerConnectWithSignerRequest) Destroy(value ConnectWithSignerRequest) {
	value.Destroy()
}

type CreateBolt12InvoiceRequest struct {
	Offer          string
	InvoiceRequest string
}

func (r *CreateBolt12InvoiceRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Offer)
	FfiDestroyerString{}.Destroy(r.InvoiceRequest)
}

type FfiConverterCreateBolt12InvoiceRequest struct{}

var FfiConverterCreateBolt12InvoiceRequestINSTANCE = FfiConverterCreateBolt12InvoiceRequest{}

func (c FfiConverterCreateBolt12InvoiceRequest) Lift(rb RustBufferI) CreateBolt12InvoiceRequest {
	return LiftFromRustBuffer[CreateBolt12InvoiceRequest](c, rb)
}

func (c FfiConverterCreateBolt12InvoiceRequest) Read(reader io.Reader) CreateBolt12InvoiceRequest {
	return CreateBolt12InvoiceRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterCreateBolt12InvoiceRequest) Lower(value CreateBolt12InvoiceRequest) C.RustBuffer {
	return LowerIntoRustBuffer[CreateBolt12InvoiceRequest](c, value)
}

func (c FfiConverterCreateBolt12InvoiceRequest) Write(writer io.Writer, value CreateBolt12InvoiceRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Offer)
	FfiConverterStringINSTANCE.Write(writer, value.InvoiceRequest)
}

type FfiDestroyerCreateBolt12InvoiceRequest struct{}

func (_ FfiDestroyerCreateBolt12InvoiceRequest) Destroy(value CreateBolt12InvoiceRequest) {
	value.Destroy()
}

type CreateBolt12InvoiceResponse struct {
	Invoice string
}

func (r *CreateBolt12InvoiceResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Invoice)
}

type FfiConverterCreateBolt12InvoiceResponse struct{}

var FfiConverterCreateBolt12InvoiceResponseINSTANCE = FfiConverterCreateBolt12InvoiceResponse{}

func (c FfiConverterCreateBolt12InvoiceResponse) Lift(rb RustBufferI) CreateBolt12InvoiceResponse {
	return LiftFromRustBuffer[CreateBolt12InvoiceResponse](c, rb)
}

func (c FfiConverterCreateBolt12InvoiceResponse) Read(reader io.Reader) CreateBolt12InvoiceResponse {
	return CreateBolt12InvoiceResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterCreateBolt12InvoiceResponse) Lower(value CreateBolt12InvoiceResponse) C.RustBuffer {
	return LowerIntoRustBuffer[CreateBolt12InvoiceResponse](c, value)
}

func (c FfiConverterCreateBolt12InvoiceResponse) Write(writer io.Writer, value CreateBolt12InvoiceResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerCreateBolt12InvoiceResponse struct{}

func (_ FfiDestroyerCreateBolt12InvoiceResponse) Destroy(value CreateBolt12InvoiceResponse) {
	value.Destroy()
}

type CurrencyInfo struct {
	Name            string
	FractionSize    uint32
	Spacing         *uint32
	Symbol          *Symbol
	UniqSymbol      *Symbol
	LocalizedName   []LocalizedName
	LocaleOverrides []LocaleOverrides
}

func (r *CurrencyInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerUint32{}.Destroy(r.FractionSize)
	FfiDestroyerOptionalUint32{}.Destroy(r.Spacing)
	FfiDestroyerOptionalSymbol{}.Destroy(r.Symbol)
	FfiDestroyerOptionalSymbol{}.Destroy(r.UniqSymbol)
	FfiDestroyerSequenceLocalizedName{}.Destroy(r.LocalizedName)
	FfiDestroyerSequenceLocaleOverrides{}.Destroy(r.LocaleOverrides)
}

type FfiConverterCurrencyInfo struct{}

var FfiConverterCurrencyInfoINSTANCE = FfiConverterCurrencyInfo{}

func (c FfiConverterCurrencyInfo) Lift(rb RustBufferI) CurrencyInfo {
	return LiftFromRustBuffer[CurrencyInfo](c, rb)
}

func (c FfiConverterCurrencyInfo) Read(reader io.Reader) CurrencyInfo {
	return CurrencyInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalSymbolINSTANCE.Read(reader),
		FfiConverterOptionalSymbolINSTANCE.Read(reader),
		FfiConverterSequenceLocalizedNameINSTANCE.Read(reader),
		FfiConverterSequenceLocaleOverridesINSTANCE.Read(reader),
	}
}

func (c FfiConverterCurrencyInfo) Lower(value CurrencyInfo) C.RustBuffer {
	return LowerIntoRustBuffer[CurrencyInfo](c, value)
}

func (c FfiConverterCurrencyInfo) Write(writer io.Writer, value CurrencyInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterUint32INSTANCE.Write(writer, value.FractionSize)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterOptionalSymbolINSTANCE.Write(writer, value.Symbol)
	FfiConverterOptionalSymbolINSTANCE.Write(writer, value.UniqSymbol)
	FfiConverterSequenceLocalizedNameINSTANCE.Write(writer, value.LocalizedName)
	FfiConverterSequenceLocaleOverridesINSTANCE.Write(writer, value.LocaleOverrides)
}

type FfiDestroyerCurrencyInfo struct{}

func (_ FfiDestroyerCurrencyInfo) Destroy(value CurrencyInfo) {
	value.Destroy()
}

type ExternalInputParser struct {
	ProviderId string
	InputRegex string
	ParserUrl  string
}

func (r *ExternalInputParser) Destroy() {
	FfiDestroyerString{}.Destroy(r.ProviderId)
	FfiDestroyerString{}.Destroy(r.InputRegex)
	FfiDestroyerString{}.Destroy(r.ParserUrl)
}

type FfiConverterExternalInputParser struct{}

var FfiConverterExternalInputParserINSTANCE = FfiConverterExternalInputParser{}

func (c FfiConverterExternalInputParser) Lift(rb RustBufferI) ExternalInputParser {
	return LiftFromRustBuffer[ExternalInputParser](c, rb)
}

func (c FfiConverterExternalInputParser) Read(reader io.Reader) ExternalInputParser {
	return ExternalInputParser{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterExternalInputParser) Lower(value ExternalInputParser) C.RustBuffer {
	return LowerIntoRustBuffer[ExternalInputParser](c, value)
}

func (c FfiConverterExternalInputParser) Write(writer io.Writer, value ExternalInputParser) {
	FfiConverterStringINSTANCE.Write(writer, value.ProviderId)
	FfiConverterStringINSTANCE.Write(writer, value.InputRegex)
	FfiConverterStringINSTANCE.Write(writer, value.ParserUrl)
}

type FfiDestroyerExternalInputParser struct{}

func (_ FfiDestroyerExternalInputParser) Destroy(value ExternalInputParser) {
	value.Destroy()
}

type FetchPaymentProposedFeesRequest struct {
	SwapId string
}

func (r *FetchPaymentProposedFeesRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapId)
}

type FfiConverterFetchPaymentProposedFeesRequest struct{}

var FfiConverterFetchPaymentProposedFeesRequestINSTANCE = FfiConverterFetchPaymentProposedFeesRequest{}

func (c FfiConverterFetchPaymentProposedFeesRequest) Lift(rb RustBufferI) FetchPaymentProposedFeesRequest {
	return LiftFromRustBuffer[FetchPaymentProposedFeesRequest](c, rb)
}

func (c FfiConverterFetchPaymentProposedFeesRequest) Read(reader io.Reader) FetchPaymentProposedFeesRequest {
	return FetchPaymentProposedFeesRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterFetchPaymentProposedFeesRequest) Lower(value FetchPaymentProposedFeesRequest) C.RustBuffer {
	return LowerIntoRustBuffer[FetchPaymentProposedFeesRequest](c, value)
}

func (c FfiConverterFetchPaymentProposedFeesRequest) Write(writer io.Writer, value FetchPaymentProposedFeesRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapId)
}

type FfiDestroyerFetchPaymentProposedFeesRequest struct{}

func (_ FfiDestroyerFetchPaymentProposedFeesRequest) Destroy(value FetchPaymentProposedFeesRequest) {
	value.Destroy()
}

type FetchPaymentProposedFeesResponse struct {
	SwapId            string
	FeesSat           uint64
	PayerAmountSat    uint64
	ReceiverAmountSat uint64
}

func (r *FetchPaymentProposedFeesResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapId)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerUint64{}.Destroy(r.PayerAmountSat)
	FfiDestroyerUint64{}.Destroy(r.ReceiverAmountSat)
}

type FfiConverterFetchPaymentProposedFeesResponse struct{}

var FfiConverterFetchPaymentProposedFeesResponseINSTANCE = FfiConverterFetchPaymentProposedFeesResponse{}

func (c FfiConverterFetchPaymentProposedFeesResponse) Lift(rb RustBufferI) FetchPaymentProposedFeesResponse {
	return LiftFromRustBuffer[FetchPaymentProposedFeesResponse](c, rb)
}

func (c FfiConverterFetchPaymentProposedFeesResponse) Read(reader io.Reader) FetchPaymentProposedFeesResponse {
	return FetchPaymentProposedFeesResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterFetchPaymentProposedFeesResponse) Lower(value FetchPaymentProposedFeesResponse) C.RustBuffer {
	return LowerIntoRustBuffer[FetchPaymentProposedFeesResponse](c, value)
}

func (c FfiConverterFetchPaymentProposedFeesResponse) Write(writer io.Writer, value FetchPaymentProposedFeesResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapId)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PayerAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.ReceiverAmountSat)
}

type FfiDestroyerFetchPaymentProposedFeesResponse struct{}

func (_ FfiDestroyerFetchPaymentProposedFeesResponse) Destroy(value FetchPaymentProposedFeesResponse) {
	value.Destroy()
}

type FiatCurrency struct {
	Id   string
	Info CurrencyInfo
}

func (r *FiatCurrency) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerCurrencyInfo{}.Destroy(r.Info)
}

type FfiConverterFiatCurrency struct{}

var FfiConverterFiatCurrencyINSTANCE = FfiConverterFiatCurrency{}

func (c FfiConverterFiatCurrency) Lift(rb RustBufferI) FiatCurrency {
	return LiftFromRustBuffer[FiatCurrency](c, rb)
}

func (c FfiConverterFiatCurrency) Read(reader io.Reader) FiatCurrency {
	return FiatCurrency{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterCurrencyInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterFiatCurrency) Lower(value FiatCurrency) C.RustBuffer {
	return LowerIntoRustBuffer[FiatCurrency](c, value)
}

func (c FfiConverterFiatCurrency) Write(writer io.Writer, value FiatCurrency) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterCurrencyInfoINSTANCE.Write(writer, value.Info)
}

type FfiDestroyerFiatCurrency struct{}

func (_ FfiDestroyerFiatCurrency) Destroy(value FiatCurrency) {
	value.Destroy()
}

type GetInfoResponse struct {
	WalletInfo     WalletInfo
	BlockchainInfo BlockchainInfo
}

func (r *GetInfoResponse) Destroy() {
	FfiDestroyerWalletInfo{}.Destroy(r.WalletInfo)
	FfiDestroyerBlockchainInfo{}.Destroy(r.BlockchainInfo)
}

type FfiConverterGetInfoResponse struct{}

var FfiConverterGetInfoResponseINSTANCE = FfiConverterGetInfoResponse{}

func (c FfiConverterGetInfoResponse) Lift(rb RustBufferI) GetInfoResponse {
	return LiftFromRustBuffer[GetInfoResponse](c, rb)
}

func (c FfiConverterGetInfoResponse) Read(reader io.Reader) GetInfoResponse {
	return GetInfoResponse{
		FfiConverterWalletInfoINSTANCE.Read(reader),
		FfiConverterBlockchainInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterGetInfoResponse) Lower(value GetInfoResponse) C.RustBuffer {
	return LowerIntoRustBuffer[GetInfoResponse](c, value)
}

func (c FfiConverterGetInfoResponse) Write(writer io.Writer, value GetInfoResponse) {
	FfiConverterWalletInfoINSTANCE.Write(writer, value.WalletInfo)
	FfiConverterBlockchainInfoINSTANCE.Write(writer, value.BlockchainInfo)
}

type FfiDestroyerGetInfoResponse struct{}

func (_ FfiDestroyerGetInfoResponse) Destroy(value GetInfoResponse) {
	value.Destroy()
}

// ///////////////////////////////
type LnInvoice struct {
	Bolt11                  string
	Network                 Network
	PayeePubkey             string
	PaymentHash             string
	Description             *string
	DescriptionHash         *string
	AmountMsat              *uint64
	Timestamp               uint64
	Expiry                  uint64
	RoutingHints            []RouteHint
	PaymentSecret           []uint8
	MinFinalCltvExpiryDelta uint64
}

func (r *LnInvoice) Destroy() {
	FfiDestroyerString{}.Destroy(r.Bolt11)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerString{}.Destroy(r.PayeePubkey)
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.DescriptionHash)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
	FfiDestroyerSequenceRouteHint{}.Destroy(r.RoutingHints)
	FfiDestroyerSequenceUint8{}.Destroy(r.PaymentSecret)
	FfiDestroyerUint64{}.Destroy(r.MinFinalCltvExpiryDelta)
}

type FfiConverterLnInvoice struct{}

var FfiConverterLnInvoiceINSTANCE = FfiConverterLnInvoice{}

func (c FfiConverterLnInvoice) Lift(rb RustBufferI) LnInvoice {
	return LiftFromRustBuffer[LnInvoice](c, rb)
}

func (c FfiConverterLnInvoice) Read(reader io.Reader) LnInvoice {
	return LnInvoice{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceRouteHintINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterLnInvoice) Lower(value LnInvoice) C.RustBuffer {
	return LowerIntoRustBuffer[LnInvoice](c, value)
}

func (c FfiConverterLnInvoice) Write(writer io.Writer, value LnInvoice) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterStringINSTANCE.Write(writer, value.PayeePubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DescriptionHash)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
	FfiConverterSequenceRouteHintINSTANCE.Write(writer, value.RoutingHints)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PaymentSecret)
	FfiConverterUint64INSTANCE.Write(writer, value.MinFinalCltvExpiryDelta)
}

type FfiDestroyerLnInvoice struct{}

func (_ FfiDestroyerLnInvoice) Destroy(value LnInvoice) {
	value.Destroy()
}

type LnOffer struct {
	Offer          string
	Chains         []string
	Paths          []LnOfferBlindedPath
	Description    *string
	SigningPubkey  *string
	MinAmount      *Amount
	AbsoluteExpiry *uint64
	Issuer         *string
}

func (r *LnOffer) Destroy() {
	FfiDestroyerString{}.Destroy(r.Offer)
	FfiDestroyerSequenceString{}.Destroy(r.Chains)
	FfiDestroyerSequenceLnOfferBlindedPath{}.Destroy(r.Paths)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.SigningPubkey)
	FfiDestroyerOptionalAmount{}.Destroy(r.MinAmount)
	FfiDestroyerOptionalUint64{}.Destroy(r.AbsoluteExpiry)
	FfiDestroyerOptionalString{}.Destroy(r.Issuer)
}

type FfiConverterLnOffer struct{}

var FfiConverterLnOfferINSTANCE = FfiConverterLnOffer{}

func (c FfiConverterLnOffer) Lift(rb RustBufferI) LnOffer {
	return LiftFromRustBuffer[LnOffer](c, rb)
}

func (c FfiConverterLnOffer) Read(reader io.Reader) LnOffer {
	return LnOffer{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceLnOfferBlindedPathINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnOffer) Lower(value LnOffer) C.RustBuffer {
	return LowerIntoRustBuffer[LnOffer](c, value)
}

func (c FfiConverterLnOffer) Write(writer io.Writer, value LnOffer) {
	FfiConverterStringINSTANCE.Write(writer, value.Offer)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.Chains)
	FfiConverterSequenceLnOfferBlindedPathINSTANCE.Write(writer, value.Paths)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.SigningPubkey)
	FfiConverterOptionalAmountINSTANCE.Write(writer, value.MinAmount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AbsoluteExpiry)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Issuer)
}

type FfiDestroyerLnOffer struct{}

func (_ FfiDestroyerLnOffer) Destroy(value LnOffer) {
	value.Destroy()
}

type LightningPaymentLimitsResponse struct {
	Send    Limits
	Receive Limits
}

func (r *LightningPaymentLimitsResponse) Destroy() {
	FfiDestroyerLimits{}.Destroy(r.Send)
	FfiDestroyerLimits{}.Destroy(r.Receive)
}

type FfiConverterLightningPaymentLimitsResponse struct{}

var FfiConverterLightningPaymentLimitsResponseINSTANCE = FfiConverterLightningPaymentLimitsResponse{}

func (c FfiConverterLightningPaymentLimitsResponse) Lift(rb RustBufferI) LightningPaymentLimitsResponse {
	return LiftFromRustBuffer[LightningPaymentLimitsResponse](c, rb)
}

func (c FfiConverterLightningPaymentLimitsResponse) Read(reader io.Reader) LightningPaymentLimitsResponse {
	return LightningPaymentLimitsResponse{
		FfiConverterLimitsINSTANCE.Read(reader),
		FfiConverterLimitsINSTANCE.Read(reader),
	}
}

func (c FfiConverterLightningPaymentLimitsResponse) Lower(value LightningPaymentLimitsResponse) C.RustBuffer {
	return LowerIntoRustBuffer[LightningPaymentLimitsResponse](c, value)
}

func (c FfiConverterLightningPaymentLimitsResponse) Write(writer io.Writer, value LightningPaymentLimitsResponse) {
	FfiConverterLimitsINSTANCE.Write(writer, value.Send)
	FfiConverterLimitsINSTANCE.Write(writer, value.Receive)
}

type FfiDestroyerLightningPaymentLimitsResponse struct{}

func (_ FfiDestroyerLightningPaymentLimitsResponse) Destroy(value LightningPaymentLimitsResponse) {
	value.Destroy()
}

type Limits struct {
	MinSat         uint64
	MaxSat         uint64
	MaxZeroConfSat uint64
}

func (r *Limits) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.MinSat)
	FfiDestroyerUint64{}.Destroy(r.MaxSat)
	FfiDestroyerUint64{}.Destroy(r.MaxZeroConfSat)
}

type FfiConverterLimits struct{}

var FfiConverterLimitsINSTANCE = FfiConverterLimits{}

func (c FfiConverterLimits) Lift(rb RustBufferI) Limits {
	return LiftFromRustBuffer[Limits](c, rb)
}

func (c FfiConverterLimits) Read(reader io.Reader) Limits {
	return Limits{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterLimits) Lower(value Limits) C.RustBuffer {
	return LowerIntoRustBuffer[Limits](c, value)
}

func (c FfiConverterLimits) Write(writer io.Writer, value Limits) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxZeroConfSat)
}

type FfiDestroyerLimits struct{}

func (_ FfiDestroyerLimits) Destroy(value Limits) {
	value.Destroy()
}

type LiquidAddressData struct {
	Address   string
	Network   Network
	AssetId   *string
	Amount    *float64
	AmountSat *uint64
	Label     *string
	Message   *string
}

func (r *LiquidAddressData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalString{}.Destroy(r.AssetId)
	FfiDestroyerOptionalFloat64{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterLiquidAddressData struct{}

var FfiConverterLiquidAddressDataINSTANCE = FfiConverterLiquidAddressData{}

func (c FfiConverterLiquidAddressData) Lift(rb RustBufferI) LiquidAddressData {
	return LiftFromRustBuffer[LiquidAddressData](c, rb)
}

func (c FfiConverterLiquidAddressData) Read(reader io.Reader) LiquidAddressData {
	return LiquidAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLiquidAddressData) Lower(value LiquidAddressData) C.RustBuffer {
	return LowerIntoRustBuffer[LiquidAddressData](c, value)
}

func (c FfiConverterLiquidAddressData) Write(writer io.Writer, value LiquidAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerLiquidAddressData struct{}

func (_ FfiDestroyerLiquidAddressData) Destroy(value LiquidAddressData) {
	value.Destroy()
}

type ListPaymentsRequest struct {
	Filters       *[]PaymentType
	States        *[]PaymentState
	FromTimestamp *int64
	ToTimestamp   *int64
	Offset        *uint32
	Limit         *uint32
	Details       *ListPaymentDetails
	SortAscending *bool
}

func (r *ListPaymentsRequest) Destroy() {
	FfiDestroyerOptionalSequencePaymentType{}.Destroy(r.Filters)
	FfiDestroyerOptionalSequencePaymentState{}.Destroy(r.States)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
	FfiDestroyerOptionalListPaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalBool{}.Destroy(r.SortAscending)
}

type FfiConverterListPaymentsRequest struct{}

var FfiConverterListPaymentsRequestINSTANCE = FfiConverterListPaymentsRequest{}

func (c FfiConverterListPaymentsRequest) Lift(rb RustBufferI) ListPaymentsRequest {
	return LiftFromRustBuffer[ListPaymentsRequest](c, rb)
}

func (c FfiConverterListPaymentsRequest) Read(reader io.Reader) ListPaymentsRequest {
	return ListPaymentsRequest{
		FfiConverterOptionalSequencePaymentTypeINSTANCE.Read(reader),
		FfiConverterOptionalSequencePaymentStateINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalListPaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterListPaymentsRequest) Lower(value ListPaymentsRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsRequest](c, value)
}

func (c FfiConverterListPaymentsRequest) Write(writer io.Writer, value ListPaymentsRequest) {
	FfiConverterOptionalSequencePaymentTypeINSTANCE.Write(writer, value.Filters)
	FfiConverterOptionalSequencePaymentStateINSTANCE.Write(writer, value.States)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
	FfiConverterOptionalListPaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.SortAscending)
}

type FfiDestroyerListPaymentsRequest struct{}

func (_ FfiDestroyerListPaymentsRequest) Destroy(value ListPaymentsRequest) {
	value.Destroy()
}

type LnOfferBlindedPath struct {
	BlindedHops []string
}

func (r *LnOfferBlindedPath) Destroy() {
	FfiDestroyerSequenceString{}.Destroy(r.BlindedHops)
}

type FfiConverterLnOfferBlindedPath struct{}

var FfiConverterLnOfferBlindedPathINSTANCE = FfiConverterLnOfferBlindedPath{}

func (c FfiConverterLnOfferBlindedPath) Lift(rb RustBufferI) LnOfferBlindedPath {
	return LiftFromRustBuffer[LnOfferBlindedPath](c, rb)
}

func (c FfiConverterLnOfferBlindedPath) Read(reader io.Reader) LnOfferBlindedPath {
	return LnOfferBlindedPath{
		FfiConverterSequenceStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnOfferBlindedPath) Lower(value LnOfferBlindedPath) C.RustBuffer {
	return LowerIntoRustBuffer[LnOfferBlindedPath](c, value)
}

func (c FfiConverterLnOfferBlindedPath) Write(writer io.Writer, value LnOfferBlindedPath) {
	FfiConverterSequenceStringINSTANCE.Write(writer, value.BlindedHops)
}

type FfiDestroyerLnOfferBlindedPath struct{}

func (_ FfiDestroyerLnOfferBlindedPath) Destroy(value LnOfferBlindedPath) {
	value.Destroy()
}

type LnUrlAuthRequestData struct {
	K1     string
	Domain string
	Url    string
	Action *string
}

func (r *LnUrlAuthRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.K1)
	FfiDestroyerString{}.Destroy(r.Domain)
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerOptionalString{}.Destroy(r.Action)
}

type FfiConverterLnUrlAuthRequestData struct{}

var FfiConverterLnUrlAuthRequestDataINSTANCE = FfiConverterLnUrlAuthRequestData{}

func (c FfiConverterLnUrlAuthRequestData) Lift(rb RustBufferI) LnUrlAuthRequestData {
	return LiftFromRustBuffer[LnUrlAuthRequestData](c, rb)
}

func (c FfiConverterLnUrlAuthRequestData) Read(reader io.Reader) LnUrlAuthRequestData {
	return LnUrlAuthRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlAuthRequestData) Lower(value LnUrlAuthRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlAuthRequestData](c, value)
}

func (c FfiConverterLnUrlAuthRequestData) Write(writer io.Writer, value LnUrlAuthRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Action)
}

type FfiDestroyerLnUrlAuthRequestData struct{}

func (_ FfiDestroyerLnUrlAuthRequestData) Destroy(value LnUrlAuthRequestData) {
	value.Destroy()
}

type LnUrlErrorData struct {
	Reason string
}

func (r *LnUrlErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterLnUrlErrorData struct{}

var FfiConverterLnUrlErrorDataINSTANCE = FfiConverterLnUrlErrorData{}

func (c FfiConverterLnUrlErrorData) Lift(rb RustBufferI) LnUrlErrorData {
	return LiftFromRustBuffer[LnUrlErrorData](c, rb)
}

func (c FfiConverterLnUrlErrorData) Read(reader io.Reader) LnUrlErrorData {
	return LnUrlErrorData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlErrorData) Lower(value LnUrlErrorData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlErrorData](c, value)
}

func (c FfiConverterLnUrlErrorData) Write(writer io.Writer, value LnUrlErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerLnUrlErrorData struct{}

func (_ FfiDestroyerLnUrlErrorData) Destroy(value LnUrlErrorData) {
	value.Destroy()
}

type LnUrlInfo struct {
	LnAddress                        *string
	LnurlPayComment                  *string
	LnurlPayDomain                   *string
	LnurlPayMetadata                 *string
	LnurlPaySuccessAction            *SuccessActionProcessed
	LnurlPayUnprocessedSuccessAction *SuccessAction
	LnurlWithdrawEndpoint            *string
}

func (r *LnUrlInfo) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.LnAddress)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayComment)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayDomain)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayMetadata)
	FfiDestroyerOptionalSuccessActionProcessed{}.Destroy(r.LnurlPaySuccessAction)
	FfiDestroyerOptionalSuccessAction{}.Destroy(r.LnurlPayUnprocessedSuccessAction)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlWithdrawEndpoint)
}

type FfiConverterLnUrlInfo struct{}

var FfiConverterLnUrlInfoINSTANCE = FfiConverterLnUrlInfo{}

func (c FfiConverterLnUrlInfo) Lift(rb RustBufferI) LnUrlInfo {
	return LiftFromRustBuffer[LnUrlInfo](c, rb)
}

func (c FfiConverterLnUrlInfo) Read(reader io.Reader) LnUrlInfo {
	return LnUrlInfo{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterOptionalSuccessActionINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlInfo) Lower(value LnUrlInfo) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlInfo](c, value)
}

func (c FfiConverterLnUrlInfo) Write(writer io.Writer, value LnUrlInfo) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayComment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayDomain)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayMetadata)
	FfiConverterOptionalSuccessActionProcessedINSTANCE.Write(writer, value.LnurlPaySuccessAction)
	FfiConverterOptionalSuccessActionINSTANCE.Write(writer, value.LnurlPayUnprocessedSuccessAction)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlWithdrawEndpoint)
}

type FfiDestroyerLnUrlInfo struct{}

func (_ FfiDestroyerLnUrlInfo) Destroy(value LnUrlInfo) {
	value.Destroy()
}

type LnUrlPayErrorData struct {
	PaymentHash string
	Reason      string
}

func (r *LnUrlPayErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterLnUrlPayErrorData struct{}

var FfiConverterLnUrlPayErrorDataINSTANCE = FfiConverterLnUrlPayErrorData{}

func (c FfiConverterLnUrlPayErrorData) Lift(rb RustBufferI) LnUrlPayErrorData {
	return LiftFromRustBuffer[LnUrlPayErrorData](c, rb)
}

func (c FfiConverterLnUrlPayErrorData) Read(reader io.Reader) LnUrlPayErrorData {
	return LnUrlPayErrorData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayErrorData) Lower(value LnUrlPayErrorData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayErrorData](c, value)
}

func (c FfiConverterLnUrlPayErrorData) Write(writer io.Writer, value LnUrlPayErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerLnUrlPayErrorData struct{}

func (_ FfiDestroyerLnUrlPayErrorData) Destroy(value LnUrlPayErrorData) {
	value.Destroy()
}

type LnUrlPayRequest struct {
	PrepareResponse PrepareLnUrlPayResponse
}

func (r *LnUrlPayRequest) Destroy() {
	FfiDestroyerPrepareLnUrlPayResponse{}.Destroy(r.PrepareResponse)
}

type FfiConverterLnUrlPayRequest struct{}

var FfiConverterLnUrlPayRequestINSTANCE = FfiConverterLnUrlPayRequest{}

func (c FfiConverterLnUrlPayRequest) Lift(rb RustBufferI) LnUrlPayRequest {
	return LiftFromRustBuffer[LnUrlPayRequest](c, rb)
}

func (c FfiConverterLnUrlPayRequest) Read(reader io.Reader) LnUrlPayRequest {
	return LnUrlPayRequest{
		FfiConverterPrepareLnUrlPayResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayRequest) Lower(value LnUrlPayRequest) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequest](c, value)
}

func (c FfiConverterLnUrlPayRequest) Write(writer io.Writer, value LnUrlPayRequest) {
	FfiConverterPrepareLnUrlPayResponseINSTANCE.Write(writer, value.PrepareResponse)
}

type FfiDestroyerLnUrlPayRequest struct{}

func (_ FfiDestroyerLnUrlPayRequest) Destroy(value LnUrlPayRequest) {
	value.Destroy()
}

type LnUrlPayRequestData struct {
	Callback       string
	MinSendable    uint64
	MaxSendable    uint64
	MetadataStr    string
	CommentAllowed uint16
	Domain         string
	AllowsNostr    bool
	NostrPubkey    *string
	LnAddress      *string
}

func (r *LnUrlPayRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Callback)
	FfiDestroyerUint64{}.Destroy(r.MinSendable)
	FfiDestroyerUint64{}.Destroy(r.MaxSendable)
	FfiDestroyerString{}.Destroy(r.MetadataStr)
	FfiDestroyerUint16{}.Destroy(r.CommentAllowed)
	FfiDestroyerString{}.Destroy(r.Domain)
	FfiDestroyerBool{}.Destroy(r.AllowsNostr)
	FfiDestroyerOptionalString{}.Destroy(r.NostrPubkey)
	FfiDestroyerOptionalString{}.Destroy(r.LnAddress)
}

type FfiConverterLnUrlPayRequestData struct{}

var FfiConverterLnUrlPayRequestDataINSTANCE = FfiConverterLnUrlPayRequestData{}

func (c FfiConverterLnUrlPayRequestData) Lift(rb RustBufferI) LnUrlPayRequestData {
	return LiftFromRustBuffer[LnUrlPayRequestData](c, rb)
}

func (c FfiConverterLnUrlPayRequestData) Read(reader io.Reader) LnUrlPayRequestData {
	return LnUrlPayRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint16INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayRequestData) Lower(value LnUrlPayRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequestData](c, value)
}

func (c FfiConverterLnUrlPayRequestData) Write(writer io.Writer, value LnUrlPayRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterUint64INSTANCE.Write(writer, value.MinSendable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSendable)
	FfiConverterStringINSTANCE.Write(writer, value.MetadataStr)
	FfiConverterUint16INSTANCE.Write(writer, value.CommentAllowed)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterBoolINSTANCE.Write(writer, value.AllowsNostr)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.NostrPubkey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
}

type FfiDestroyerLnUrlPayRequestData struct{}

func (_ FfiDestroyerLnUrlPayRequestData) Destroy(value LnUrlPayRequestData) {
	value.Destroy()
}

type LnUrlPaySuccessData struct {
	SuccessAction *SuccessActionProcessed
	Payment       Payment
}

func (r *LnUrlPaySuccessData) Destroy() {
	FfiDestroyerOptionalSuccessActionProcessed{}.Destroy(r.SuccessAction)
	FfiDestroyerPayment{}.Destroy(r.Payment)
}

type FfiConverterLnUrlPaySuccessData struct{}

var FfiConverterLnUrlPaySuccessDataINSTANCE = FfiConverterLnUrlPaySuccessData{}

func (c FfiConverterLnUrlPaySuccessData) Lift(rb RustBufferI) LnUrlPaySuccessData {
	return LiftFromRustBuffer[LnUrlPaySuccessData](c, rb)
}

func (c FfiConverterLnUrlPaySuccessData) Read(reader io.Reader) LnUrlPaySuccessData {
	return LnUrlPaySuccessData{
		FfiConverterOptionalSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPaySuccessData) Lower(value LnUrlPaySuccessData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPaySuccessData](c, value)
}

func (c FfiConverterLnUrlPaySuccessData) Write(writer io.Writer, value LnUrlPaySuccessData) {
	FfiConverterOptionalSuccessActionProcessedINSTANCE.Write(writer, value.SuccessAction)
	FfiConverterPaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerLnUrlPaySuccessData struct{}

func (_ FfiDestroyerLnUrlPaySuccessData) Destroy(value LnUrlPaySuccessData) {
	value.Destroy()
}

type LnUrlWithdrawRequest struct {
	Data        LnUrlWithdrawRequestData
	AmountMsat  uint64
	Description *string
}

func (r *LnUrlWithdrawRequest) Destroy() {
	FfiDestroyerLnUrlWithdrawRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
}

type FfiConverterLnUrlWithdrawRequest struct{}

var FfiConverterLnUrlWithdrawRequestINSTANCE = FfiConverterLnUrlWithdrawRequest{}

func (c FfiConverterLnUrlWithdrawRequest) Lift(rb RustBufferI) LnUrlWithdrawRequest {
	return LiftFromRustBuffer[LnUrlWithdrawRequest](c, rb)
}

func (c FfiConverterLnUrlWithdrawRequest) Read(reader io.Reader) LnUrlWithdrawRequest {
	return LnUrlWithdrawRequest{
		FfiConverterLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawRequest) Lower(value LnUrlWithdrawRequest) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequest](c, value)
}

func (c FfiConverterLnUrlWithdrawRequest) Write(writer io.Writer, value LnUrlWithdrawRequest) {
	FfiConverterLnUrlWithdrawRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
}

type FfiDestroyerLnUrlWithdrawRequest struct{}

func (_ FfiDestroyerLnUrlWithdrawRequest) Destroy(value LnUrlWithdrawRequest) {
	value.Destroy()
}

type LnUrlWithdrawRequestData struct {
	Callback           string
	K1                 string
	DefaultDescription string
	MinWithdrawable    uint64
	MaxWithdrawable    uint64
}

func (r *LnUrlWithdrawRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Callback)
	FfiDestroyerString{}.Destroy(r.K1)
	FfiDestroyerString{}.Destroy(r.DefaultDescription)
	FfiDestroyerUint64{}.Destroy(r.MinWithdrawable)
	FfiDestroyerUint64{}.Destroy(r.MaxWithdrawable)
}

type FfiConverterLnUrlWithdrawRequestData struct{}

var FfiConverterLnUrlWithdrawRequestDataINSTANCE = FfiConverterLnUrlWithdrawRequestData{}

func (c FfiConverterLnUrlWithdrawRequestData) Lift(rb RustBufferI) LnUrlWithdrawRequestData {
	return LiftFromRustBuffer[LnUrlWithdrawRequestData](c, rb)
}

func (c FfiConverterLnUrlWithdrawRequestData) Read(reader io.Reader) LnUrlWithdrawRequestData {
	return LnUrlWithdrawRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawRequestData) Lower(value LnUrlWithdrawRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequestData](c, value)
}

func (c FfiConverterLnUrlWithdrawRequestData) Write(writer io.Writer, value LnUrlWithdrawRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.DefaultDescription)
	FfiConverterUint64INSTANCE.Write(writer, value.MinWithdrawable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxWithdrawable)
}

type FfiDestroyerLnUrlWithdrawRequestData struct{}

func (_ FfiDestroyerLnUrlWithdrawRequestData) Destroy(value LnUrlWithdrawRequestData) {
	value.Destroy()
}

type LnUrlWithdrawSuccessData struct {
	Invoice LnInvoice
}

func (r *LnUrlWithdrawSuccessData) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(r.Invoice)
}

type FfiConverterLnUrlWithdrawSuccessData struct{}

var FfiConverterLnUrlWithdrawSuccessDataINSTANCE = FfiConverterLnUrlWithdrawSuccessData{}

func (c FfiConverterLnUrlWithdrawSuccessData) Lift(rb RustBufferI) LnUrlWithdrawSuccessData {
	return LiftFromRustBuffer[LnUrlWithdrawSuccessData](c, rb)
}

func (c FfiConverterLnUrlWithdrawSuccessData) Read(reader io.Reader) LnUrlWithdrawSuccessData {
	return LnUrlWithdrawSuccessData{
		FfiConverterLnInvoiceINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawSuccessData) Lower(value LnUrlWithdrawSuccessData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawSuccessData](c, value)
}

func (c FfiConverterLnUrlWithdrawSuccessData) Write(writer io.Writer, value LnUrlWithdrawSuccessData) {
	FfiConverterLnInvoiceINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerLnUrlWithdrawSuccessData struct{}

func (_ FfiDestroyerLnUrlWithdrawSuccessData) Destroy(value LnUrlWithdrawSuccessData) {
	value.Destroy()
}

type LocaleOverrides struct {
	Locale  string
	Spacing *uint32
	Symbol  Symbol
}

func (r *LocaleOverrides) Destroy() {
	FfiDestroyerString{}.Destroy(r.Locale)
	FfiDestroyerOptionalUint32{}.Destroy(r.Spacing)
	FfiDestroyerSymbol{}.Destroy(r.Symbol)
}

type FfiConverterLocaleOverrides struct{}

var FfiConverterLocaleOverridesINSTANCE = FfiConverterLocaleOverrides{}

func (c FfiConverterLocaleOverrides) Lift(rb RustBufferI) LocaleOverrides {
	return LiftFromRustBuffer[LocaleOverrides](c, rb)
}

func (c FfiConverterLocaleOverrides) Read(reader io.Reader) LocaleOverrides {
	return LocaleOverrides{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterSymbolINSTANCE.Read(reader),
	}
}

func (c FfiConverterLocaleOverrides) Lower(value LocaleOverrides) C.RustBuffer {
	return LowerIntoRustBuffer[LocaleOverrides](c, value)
}

func (c FfiConverterLocaleOverrides) Write(writer io.Writer, value LocaleOverrides) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterSymbolINSTANCE.Write(writer, value.Symbol)
}

type FfiDestroyerLocaleOverrides struct{}

func (_ FfiDestroyerLocaleOverrides) Destroy(value LocaleOverrides) {
	value.Destroy()
}

type LocalizedName struct {
	Locale string
	Name   string
}

func (r *LocalizedName) Destroy() {
	FfiDestroyerString{}.Destroy(r.Locale)
	FfiDestroyerString{}.Destroy(r.Name)
}

type FfiConverterLocalizedName struct{}

var FfiConverterLocalizedNameINSTANCE = FfiConverterLocalizedName{}

func (c FfiConverterLocalizedName) Lift(rb RustBufferI) LocalizedName {
	return LiftFromRustBuffer[LocalizedName](c, rb)
}

func (c FfiConverterLocalizedName) Read(reader io.Reader) LocalizedName {
	return LocalizedName{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLocalizedName) Lower(value LocalizedName) C.RustBuffer {
	return LowerIntoRustBuffer[LocalizedName](c, value)
}

func (c FfiConverterLocalizedName) Write(writer io.Writer, value LocalizedName) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
}

type FfiDestroyerLocalizedName struct{}

func (_ FfiDestroyerLocalizedName) Destroy(value LocalizedName) {
	value.Destroy()
}

type LogEntry struct {
	Line  string
	Level string
}

func (r *LogEntry) Destroy() {
	FfiDestroyerString{}.Destroy(r.Line)
	FfiDestroyerString{}.Destroy(r.Level)
}

type FfiConverterLogEntry struct{}

var FfiConverterLogEntryINSTANCE = FfiConverterLogEntry{}

func (c FfiConverterLogEntry) Lift(rb RustBufferI) LogEntry {
	return LiftFromRustBuffer[LogEntry](c, rb)
}

func (c FfiConverterLogEntry) Read(reader io.Reader) LogEntry {
	return LogEntry{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLogEntry) Lower(value LogEntry) C.RustBuffer {
	return LowerIntoRustBuffer[LogEntry](c, value)
}

func (c FfiConverterLogEntry) Write(writer io.Writer, value LogEntry) {
	FfiConverterStringINSTANCE.Write(writer, value.Line)
	FfiConverterStringINSTANCE.Write(writer, value.Level)
}

type FfiDestroyerLogEntry struct{}

func (_ FfiDestroyerLogEntry) Destroy(value LogEntry) {
	value.Destroy()
}

type MessageSuccessActionData struct {
	Message string
}

func (r *MessageSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterMessageSuccessActionData struct{}

var FfiConverterMessageSuccessActionDataINSTANCE = FfiConverterMessageSuccessActionData{}

func (c FfiConverterMessageSuccessActionData) Lift(rb RustBufferI) MessageSuccessActionData {
	return LiftFromRustBuffer[MessageSuccessActionData](c, rb)
}

func (c FfiConverterMessageSuccessActionData) Read(reader io.Reader) MessageSuccessActionData {
	return MessageSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterMessageSuccessActionData) Lower(value MessageSuccessActionData) C.RustBuffer {
	return LowerIntoRustBuffer[MessageSuccessActionData](c, value)
}

func (c FfiConverterMessageSuccessActionData) Write(writer io.Writer, value MessageSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerMessageSuccessActionData struct{}

func (_ FfiDestroyerMessageSuccessActionData) Destroy(value MessageSuccessActionData) {
	value.Destroy()
}

type OnchainPaymentLimitsResponse struct {
	Send    Limits
	Receive Limits
}

func (r *OnchainPaymentLimitsResponse) Destroy() {
	FfiDestroyerLimits{}.Destroy(r.Send)
	FfiDestroyerLimits{}.Destroy(r.Receive)
}

type FfiConverterOnchainPaymentLimitsResponse struct{}

var FfiConverterOnchainPaymentLimitsResponseINSTANCE = FfiConverterOnchainPaymentLimitsResponse{}

func (c FfiConverterOnchainPaymentLimitsResponse) Lift(rb RustBufferI) OnchainPaymentLimitsResponse {
	return LiftFromRustBuffer[OnchainPaymentLimitsResponse](c, rb)
}

func (c FfiConverterOnchainPaymentLimitsResponse) Read(reader io.Reader) OnchainPaymentLimitsResponse {
	return OnchainPaymentLimitsResponse{
		FfiConverterLimitsINSTANCE.Read(reader),
		FfiConverterLimitsINSTANCE.Read(reader),
	}
}

func (c FfiConverterOnchainPaymentLimitsResponse) Lower(value OnchainPaymentLimitsResponse) C.RustBuffer {
	return LowerIntoRustBuffer[OnchainPaymentLimitsResponse](c, value)
}

func (c FfiConverterOnchainPaymentLimitsResponse) Write(writer io.Writer, value OnchainPaymentLimitsResponse) {
	FfiConverterLimitsINSTANCE.Write(writer, value.Send)
	FfiConverterLimitsINSTANCE.Write(writer, value.Receive)
}

type FfiDestroyerOnchainPaymentLimitsResponse struct{}

func (_ FfiDestroyerOnchainPaymentLimitsResponse) Destroy(value OnchainPaymentLimitsResponse) {
	value.Destroy()
}

type PayOnchainRequest struct {
	Address         string
	PrepareResponse PreparePayOnchainResponse
}

func (r *PayOnchainRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerPreparePayOnchainResponse{}.Destroy(r.PrepareResponse)
}

type FfiConverterPayOnchainRequest struct{}

var FfiConverterPayOnchainRequestINSTANCE = FfiConverterPayOnchainRequest{}

func (c FfiConverterPayOnchainRequest) Lift(rb RustBufferI) PayOnchainRequest {
	return LiftFromRustBuffer[PayOnchainRequest](c, rb)
}

func (c FfiConverterPayOnchainRequest) Read(reader io.Reader) PayOnchainRequest {
	return PayOnchainRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterPreparePayOnchainResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterPayOnchainRequest) Lower(value PayOnchainRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PayOnchainRequest](c, value)
}

func (c FfiConverterPayOnchainRequest) Write(writer io.Writer, value PayOnchainRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterPreparePayOnchainResponseINSTANCE.Write(writer, value.PrepareResponse)
}

type FfiDestroyerPayOnchainRequest struct{}

func (_ FfiDestroyerPayOnchainRequest) Destroy(value PayOnchainRequest) {
	value.Destroy()
}

type Payment struct {
	Timestamp      uint32
	AmountSat      uint64
	FeesSat        uint64
	PaymentType    PaymentType
	Status         PaymentState
	Details        PaymentDetails
	SwapperFeesSat *uint64
	Destination    *string
	TxId           *string
	UnblindingData *string
}

func (r *Payment) Destroy() {
	FfiDestroyerUint32{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerPaymentType{}.Destroy(r.PaymentType)
	FfiDestroyerPaymentState{}.Destroy(r.Status)
	FfiDestroyerPaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalUint64{}.Destroy(r.SwapperFeesSat)
	FfiDestroyerOptionalString{}.Destroy(r.Destination)
	FfiDestroyerOptionalString{}.Destroy(r.TxId)
	FfiDestroyerOptionalString{}.Destroy(r.UnblindingData)
}

type FfiConverterPayment struct{}

var FfiConverterPaymentINSTANCE = FfiConverterPayment{}

func (c FfiConverterPayment) Lift(rb RustBufferI) Payment {
	return LiftFromRustBuffer[Payment](c, rb)
}

func (c FfiConverterPayment) Read(reader io.Reader) Payment {
	return Payment{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterPaymentTypeINSTANCE.Read(reader),
		FfiConverterPaymentStateINSTANCE.Read(reader),
		FfiConverterPaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterPayment) Lower(value Payment) C.RustBuffer {
	return LowerIntoRustBuffer[Payment](c, value)
}

func (c FfiConverterPayment) Write(writer io.Writer, value Payment) {
	FfiConverterUint32INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterPaymentTypeINSTANCE.Write(writer, value.PaymentType)
	FfiConverterPaymentStateINSTANCE.Write(writer, value.Status)
	FfiConverterPaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.SwapperFeesSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.TxId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.UnblindingData)
}

type FfiDestroyerPayment struct{}

func (_ FfiDestroyerPayment) Destroy(value Payment) {
	value.Destroy()
}

type PrepareBuyBitcoinRequest struct {
	Provider  BuyBitcoinProvider
	AmountSat uint64
}

func (r *PrepareBuyBitcoinRequest) Destroy() {
	FfiDestroyerBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
}

type FfiConverterPrepareBuyBitcoinRequest struct{}

var FfiConverterPrepareBuyBitcoinRequestINSTANCE = FfiConverterPrepareBuyBitcoinRequest{}

func (c FfiConverterPrepareBuyBitcoinRequest) Lift(rb RustBufferI) PrepareBuyBitcoinRequest {
	return LiftFromRustBuffer[PrepareBuyBitcoinRequest](c, rb)
}

func (c FfiConverterPrepareBuyBitcoinRequest) Read(reader io.Reader) PrepareBuyBitcoinRequest {
	return PrepareBuyBitcoinRequest{
		FfiConverterBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareBuyBitcoinRequest) Lower(value PrepareBuyBitcoinRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareBuyBitcoinRequest](c, value)
}

func (c FfiConverterPrepareBuyBitcoinRequest) Write(writer io.Writer, value PrepareBuyBitcoinRequest) {
	FfiConverterBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
}

type FfiDestroyerPrepareBuyBitcoinRequest struct{}

func (_ FfiDestroyerPrepareBuyBitcoinRequest) Destroy(value PrepareBuyBitcoinRequest) {
	value.Destroy()
}

type PrepareBuyBitcoinResponse struct {
	Provider  BuyBitcoinProvider
	AmountSat uint64
	FeesSat   uint64
}

func (r *PrepareBuyBitcoinResponse) Destroy() {
	FfiDestroyerBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
}

type FfiConverterPrepareBuyBitcoinResponse struct{}

var FfiConverterPrepareBuyBitcoinResponseINSTANCE = FfiConverterPrepareBuyBitcoinResponse{}

func (c FfiConverterPrepareBuyBitcoinResponse) Lift(rb RustBufferI) PrepareBuyBitcoinResponse {
	return LiftFromRustBuffer[PrepareBuyBitcoinResponse](c, rb)
}

func (c FfiConverterPrepareBuyBitcoinResponse) Read(reader io.Reader) PrepareBuyBitcoinResponse {
	return PrepareBuyBitcoinResponse{
		FfiConverterBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareBuyBitcoinResponse) Lower(value PrepareBuyBitcoinResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareBuyBitcoinResponse](c, value)
}

func (c FfiConverterPrepareBuyBitcoinResponse) Write(writer io.Writer, value PrepareBuyBitcoinResponse) {
	FfiConverterBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
}

type FfiDestroyerPrepareBuyBitcoinResponse struct{}

func (_ FfiDestroyerPrepareBuyBitcoinResponse) Destroy(value PrepareBuyBitcoinResponse) {
	value.Destroy()
}

type PrepareLnUrlPayRequest struct {
	Data                     LnUrlPayRequestData
	Amount                   PayAmount
	Bip353Address            *string
	Comment                  *string
	ValidateSuccessActionUrl *bool
}

func (r *PrepareLnUrlPayRequest) Destroy() {
	FfiDestroyerLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerPayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalString{}.Destroy(r.Bip353Address)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalBool{}.Destroy(r.ValidateSuccessActionUrl)
}

type FfiConverterPrepareLnUrlPayRequest struct{}

var FfiConverterPrepareLnUrlPayRequestINSTANCE = FfiConverterPrepareLnUrlPayRequest{}

func (c FfiConverterPrepareLnUrlPayRequest) Lift(rb RustBufferI) PrepareLnUrlPayRequest {
	return LiftFromRustBuffer[PrepareLnUrlPayRequest](c, rb)
}

func (c FfiConverterPrepareLnUrlPayRequest) Read(reader io.Reader) PrepareLnUrlPayRequest {
	return PrepareLnUrlPayRequest{
		FfiConverterLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterPayAmountINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareLnUrlPayRequest) Lower(value PrepareLnUrlPayRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareLnUrlPayRequest](c, value)
}

func (c FfiConverterPrepareLnUrlPayRequest) Write(writer io.Writer, value PrepareLnUrlPayRequest) {
	FfiConverterLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterPayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bip353Address)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.ValidateSuccessActionUrl)
}

type FfiDestroyerPrepareLnUrlPayRequest struct{}

func (_ FfiDestroyerPrepareLnUrlPayRequest) Destroy(value PrepareLnUrlPayRequest) {
	value.Destroy()
}

type PrepareLnUrlPayResponse struct {
	Destination   SendDestination
	FeesSat       uint64
	Data          LnUrlPayRequestData
	Amount        PayAmount
	Comment       *string
	SuccessAction *SuccessAction
}

func (r *PrepareLnUrlPayResponse) Destroy() {
	FfiDestroyerSendDestination{}.Destroy(r.Destination)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerPayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalSuccessAction{}.Destroy(r.SuccessAction)
}

type FfiConverterPrepareLnUrlPayResponse struct{}

var FfiConverterPrepareLnUrlPayResponseINSTANCE = FfiConverterPrepareLnUrlPayResponse{}

func (c FfiConverterPrepareLnUrlPayResponse) Lift(rb RustBufferI) PrepareLnUrlPayResponse {
	return LiftFromRustBuffer[PrepareLnUrlPayResponse](c, rb)
}

func (c FfiConverterPrepareLnUrlPayResponse) Read(reader io.Reader) PrepareLnUrlPayResponse {
	return PrepareLnUrlPayResponse{
		FfiConverterSendDestinationINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterPayAmountINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalSuccessActionINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareLnUrlPayResponse) Lower(value PrepareLnUrlPayResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareLnUrlPayResponse](c, value)
}

func (c FfiConverterPrepareLnUrlPayResponse) Write(writer io.Writer, value PrepareLnUrlPayResponse) {
	FfiConverterSendDestinationINSTANCE.Write(writer, value.Destination)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterPayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalSuccessActionINSTANCE.Write(writer, value.SuccessAction)
}

type FfiDestroyerPrepareLnUrlPayResponse struct{}

func (_ FfiDestroyerPrepareLnUrlPayResponse) Destroy(value PrepareLnUrlPayResponse) {
	value.Destroy()
}

type PreparePayOnchainRequest struct {
	Amount             PayAmount
	FeeRateSatPerVbyte *uint32
}

func (r *PreparePayOnchainRequest) Destroy() {
	FfiDestroyerPayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint32{}.Destroy(r.FeeRateSatPerVbyte)
}

type FfiConverterPreparePayOnchainRequest struct{}

var FfiConverterPreparePayOnchainRequestINSTANCE = FfiConverterPreparePayOnchainRequest{}

func (c FfiConverterPreparePayOnchainRequest) Lift(rb RustBufferI) PreparePayOnchainRequest {
	return LiftFromRustBuffer[PreparePayOnchainRequest](c, rb)
}

func (c FfiConverterPreparePayOnchainRequest) Read(reader io.Reader) PreparePayOnchainRequest {
	return PreparePayOnchainRequest{
		FfiConverterPayAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterPreparePayOnchainRequest) Lower(value PreparePayOnchainRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PreparePayOnchainRequest](c, value)
}

func (c FfiConverterPreparePayOnchainRequest) Write(writer io.Writer, value PreparePayOnchainRequest) {
	FfiConverterPayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerPreparePayOnchainRequest struct{}

func (_ FfiDestroyerPreparePayOnchainRequest) Destroy(value PreparePayOnchainRequest) {
	value.Destroy()
}

type PreparePayOnchainResponse struct {
	ReceiverAmountSat uint64
	ClaimFeesSat      uint64
	TotalFeesSat      uint64
}

func (r *PreparePayOnchainResponse) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.ReceiverAmountSat)
	FfiDestroyerUint64{}.Destroy(r.ClaimFeesSat)
	FfiDestroyerUint64{}.Destroy(r.TotalFeesSat)
}

type FfiConverterPreparePayOnchainResponse struct{}

var FfiConverterPreparePayOnchainResponseINSTANCE = FfiConverterPreparePayOnchainResponse{}

func (c FfiConverterPreparePayOnchainResponse) Lift(rb RustBufferI) PreparePayOnchainResponse {
	return LiftFromRustBuffer[PreparePayOnchainResponse](c, rb)
}

func (c FfiConverterPreparePayOnchainResponse) Read(reader io.Reader) PreparePayOnchainResponse {
	return PreparePayOnchainResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPreparePayOnchainResponse) Lower(value PreparePayOnchainResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PreparePayOnchainResponse](c, value)
}

func (c FfiConverterPreparePayOnchainResponse) Write(writer io.Writer, value PreparePayOnchainResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.ReceiverAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.ClaimFeesSat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalFeesSat)
}

type FfiDestroyerPreparePayOnchainResponse struct{}

func (_ FfiDestroyerPreparePayOnchainResponse) Destroy(value PreparePayOnchainResponse) {
	value.Destroy()
}

type PrepareReceiveRequest struct {
	PaymentMethod PaymentMethod
	Amount        *ReceiveAmount
}

func (r *PrepareReceiveRequest) Destroy() {
	FfiDestroyerPaymentMethod{}.Destroy(r.PaymentMethod)
	FfiDestroyerOptionalReceiveAmount{}.Destroy(r.Amount)
}

type FfiConverterPrepareReceiveRequest struct{}

var FfiConverterPrepareReceiveRequestINSTANCE = FfiConverterPrepareReceiveRequest{}

func (c FfiConverterPrepareReceiveRequest) Lift(rb RustBufferI) PrepareReceiveRequest {
	return LiftFromRustBuffer[PrepareReceiveRequest](c, rb)
}

func (c FfiConverterPrepareReceiveRequest) Read(reader io.Reader) PrepareReceiveRequest {
	return PrepareReceiveRequest{
		FfiConverterPaymentMethodINSTANCE.Read(reader),
		FfiConverterOptionalReceiveAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareReceiveRequest) Lower(value PrepareReceiveRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveRequest](c, value)
}

func (c FfiConverterPrepareReceiveRequest) Write(writer io.Writer, value PrepareReceiveRequest) {
	FfiConverterPaymentMethodINSTANCE.Write(writer, value.PaymentMethod)
	FfiConverterOptionalReceiveAmountINSTANCE.Write(writer, value.Amount)
}

type FfiDestroyerPrepareReceiveRequest struct{}

func (_ FfiDestroyerPrepareReceiveRequest) Destroy(value PrepareReceiveRequest) {
	value.Destroy()
}

type PrepareReceiveResponse struct {
	PaymentMethod     PaymentMethod
	FeesSat           uint64
	Amount            *ReceiveAmount
	MinPayerAmountSat *uint64
	MaxPayerAmountSat *uint64
	SwapperFeerate    *float64
}

func (r *PrepareReceiveResponse) Destroy() {
	FfiDestroyerPaymentMethod{}.Destroy(r.PaymentMethod)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerOptionalReceiveAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint64{}.Destroy(r.MinPayerAmountSat)
	FfiDestroyerOptionalUint64{}.Destroy(r.MaxPayerAmountSat)
	FfiDestroyerOptionalFloat64{}.Destroy(r.SwapperFeerate)
}

type FfiConverterPrepareReceiveResponse struct{}

var FfiConverterPrepareReceiveResponseINSTANCE = FfiConverterPrepareReceiveResponse{}

func (c FfiConverterPrepareReceiveResponse) Lift(rb RustBufferI) PrepareReceiveResponse {
	return LiftFromRustBuffer[PrepareReceiveResponse](c, rb)
}

func (c FfiConverterPrepareReceiveResponse) Read(reader io.Reader) PrepareReceiveResponse {
	return PrepareReceiveResponse{
		FfiConverterPaymentMethodINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalReceiveAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareReceiveResponse) Lower(value PrepareReceiveResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveResponse](c, value)
}

func (c FfiConverterPrepareReceiveResponse) Write(writer io.Writer, value PrepareReceiveResponse) {
	FfiConverterPaymentMethodINSTANCE.Write(writer, value.PaymentMethod)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterOptionalReceiveAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.MinPayerAmountSat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.MaxPayerAmountSat)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.SwapperFeerate)
}

type FfiDestroyerPrepareReceiveResponse struct{}

func (_ FfiDestroyerPrepareReceiveResponse) Destroy(value PrepareReceiveResponse) {
	value.Destroy()
}

type PrepareRefundRequest struct {
	SwapAddress        string
	RefundAddress      string
	FeeRateSatPerVbyte uint32
}

func (r *PrepareRefundRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapAddress)
	FfiDestroyerString{}.Destroy(r.RefundAddress)
	FfiDestroyerUint32{}.Destroy(r.FeeRateSatPerVbyte)
}

type FfiConverterPrepareRefundRequest struct{}

var FfiConverterPrepareRefundRequestINSTANCE = FfiConverterPrepareRefundRequest{}

func (c FfiConverterPrepareRefundRequest) Lift(rb RustBufferI) PrepareRefundRequest {
	return LiftFromRustBuffer[PrepareRefundRequest](c, rb)
}

func (c FfiConverterPrepareRefundRequest) Read(reader io.Reader) PrepareRefundRequest {
	return PrepareRefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRefundRequest) Lower(value PrepareRefundRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundRequest](c, value)
}

func (c FfiConverterPrepareRefundRequest) Write(writer io.Writer, value PrepareRefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.RefundAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerPrepareRefundRequest struct{}

func (_ FfiDestroyerPrepareRefundRequest) Destroy(value PrepareRefundRequest) {
	value.Destroy()
}

type PrepareRefundResponse struct {
	TxVsize        uint32
	TxFeeSat       uint64
	LastRefundTxId *string
}

func (r *PrepareRefundResponse) Destroy() {
	FfiDestroyerUint32{}.Destroy(r.TxVsize)
	FfiDestroyerUint64{}.Destroy(r.TxFeeSat)
	FfiDestroyerOptionalString{}.Destroy(r.LastRefundTxId)
}

type FfiConverterPrepareRefundResponse struct{}

var FfiConverterPrepareRefundResponseINSTANCE = FfiConverterPrepareRefundResponse{}

func (c FfiConverterPrepareRefundResponse) Lift(rb RustBufferI) PrepareRefundResponse {
	return LiftFromRustBuffer[PrepareRefundResponse](c, rb)
}

func (c FfiConverterPrepareRefundResponse) Read(reader io.Reader) PrepareRefundResponse {
	return PrepareRefundResponse{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRefundResponse) Lower(value PrepareRefundResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundResponse](c, value)
}

func (c FfiConverterPrepareRefundResponse) Write(writer io.Writer, value PrepareRefundResponse) {
	FfiConverterUint32INSTANCE.Write(writer, value.TxVsize)
	FfiConverterUint64INSTANCE.Write(writer, value.TxFeeSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRefundTxId)
}

type FfiDestroyerPrepareRefundResponse struct{}

func (_ FfiDestroyerPrepareRefundResponse) Destroy(value PrepareRefundResponse) {
	value.Destroy()
}

type PrepareSendRequest struct {
	Destination string
	Amount      *PayAmount
}

func (r *PrepareSendRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Destination)
	FfiDestroyerOptionalPayAmount{}.Destroy(r.Amount)
}

type FfiConverterPrepareSendRequest struct{}

var FfiConverterPrepareSendRequestINSTANCE = FfiConverterPrepareSendRequest{}

func (c FfiConverterPrepareSendRequest) Lift(rb RustBufferI) PrepareSendRequest {
	return LiftFromRustBuffer[PrepareSendRequest](c, rb)
}

func (c FfiConverterPrepareSendRequest) Read(reader io.Reader) PrepareSendRequest {
	return PrepareSendRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalPayAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareSendRequest) Lower(value PrepareSendRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareSendRequest](c, value)
}

func (c FfiConverterPrepareSendRequest) Write(writer io.Writer, value PrepareSendRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalPayAmountINSTANCE.Write(writer, value.Amount)
}

type FfiDestroyerPrepareSendRequest struct{}

func (_ FfiDestroyerPrepareSendRequest) Destroy(value PrepareSendRequest) {
	value.Destroy()
}

type PrepareSendResponse struct {
	Destination        SendDestination
	Amount             *PayAmount
	FeesSat            *uint64
	EstimatedAssetFees *float64
}

func (r *PrepareSendResponse) Destroy() {
	FfiDestroyerSendDestination{}.Destroy(r.Destination)
	FfiDestroyerOptionalPayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint64{}.Destroy(r.FeesSat)
	FfiDestroyerOptionalFloat64{}.Destroy(r.EstimatedAssetFees)
}

type FfiConverterPrepareSendResponse struct{}

var FfiConverterPrepareSendResponseINSTANCE = FfiConverterPrepareSendResponse{}

func (c FfiConverterPrepareSendResponse) Lift(rb RustBufferI) PrepareSendResponse {
	return LiftFromRustBuffer[PrepareSendResponse](c, rb)
}

func (c FfiConverterPrepareSendResponse) Read(reader io.Reader) PrepareSendResponse {
	return PrepareSendResponse{
		FfiConverterSendDestinationINSTANCE.Read(reader),
		FfiConverterOptionalPayAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareSendResponse) Lower(value PrepareSendResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareSendResponse](c, value)
}

func (c FfiConverterPrepareSendResponse) Write(writer io.Writer, value PrepareSendResponse) {
	FfiConverterSendDestinationINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalPayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.EstimatedAssetFees)
}

type FfiDestroyerPrepareSendResponse struct{}

func (_ FfiDestroyerPrepareSendResponse) Destroy(value PrepareSendResponse) {
	value.Destroy()
}

type Rate struct {
	Coin  string
	Value float64
}

func (r *Rate) Destroy() {
	FfiDestroyerString{}.Destroy(r.Coin)
	FfiDestroyerFloat64{}.Destroy(r.Value)
}

type FfiConverterRate struct{}

var FfiConverterRateINSTANCE = FfiConverterRate{}

func (c FfiConverterRate) Lift(rb RustBufferI) Rate {
	return LiftFromRustBuffer[Rate](c, rb)
}

func (c FfiConverterRate) Read(reader io.Reader) Rate {
	return Rate{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRate) Lower(value Rate) C.RustBuffer {
	return LowerIntoRustBuffer[Rate](c, value)
}

func (c FfiConverterRate) Write(writer io.Writer, value Rate) {
	FfiConverterStringINSTANCE.Write(writer, value.Coin)
	FfiConverterFloat64INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerRate struct{}

func (_ FfiDestroyerRate) Destroy(value Rate) {
	value.Destroy()
}

type ReceivePaymentRequest struct {
	PrepareResponse    PrepareReceiveResponse
	Description        *string
	UseDescriptionHash *bool
	PayerNote          *string
}

func (r *ReceivePaymentRequest) Destroy() {
	FfiDestroyerPrepareReceiveResponse{}.Destroy(r.PrepareResponse)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalBool{}.Destroy(r.UseDescriptionHash)
	FfiDestroyerOptionalString{}.Destroy(r.PayerNote)
}

type FfiConverterReceivePaymentRequest struct{}

var FfiConverterReceivePaymentRequestINSTANCE = FfiConverterReceivePaymentRequest{}

func (c FfiConverterReceivePaymentRequest) Lift(rb RustBufferI) ReceivePaymentRequest {
	return LiftFromRustBuffer[ReceivePaymentRequest](c, rb)
}

func (c FfiConverterReceivePaymentRequest) Read(reader io.Reader) ReceivePaymentRequest {
	return ReceivePaymentRequest{
		FfiConverterPrepareReceiveResponseINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterReceivePaymentRequest) Lower(value ReceivePaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentRequest](c, value)
}

func (c FfiConverterReceivePaymentRequest) Write(writer io.Writer, value ReceivePaymentRequest) {
	FfiConverterPrepareReceiveResponseINSTANCE.Write(writer, value.PrepareResponse)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.UseDescriptionHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PayerNote)
}

type FfiDestroyerReceivePaymentRequest struct{}

func (_ FfiDestroyerReceivePaymentRequest) Destroy(value ReceivePaymentRequest) {
	value.Destroy()
}

type ReceivePaymentResponse struct {
	Destination string
}

func (r *ReceivePaymentResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Destination)
}

type FfiConverterReceivePaymentResponse struct{}

var FfiConverterReceivePaymentResponseINSTANCE = FfiConverterReceivePaymentResponse{}

func (c FfiConverterReceivePaymentResponse) Lift(rb RustBufferI) ReceivePaymentResponse {
	return LiftFromRustBuffer[ReceivePaymentResponse](c, rb)
}

func (c FfiConverterReceivePaymentResponse) Read(reader io.Reader) ReceivePaymentResponse {
	return ReceivePaymentResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterReceivePaymentResponse) Lower(value ReceivePaymentResponse) C.RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentResponse](c, value)
}

func (c FfiConverterReceivePaymentResponse) Write(writer io.Writer, value ReceivePaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Destination)
}

type FfiDestroyerReceivePaymentResponse struct{}

func (_ FfiDestroyerReceivePaymentResponse) Destroy(value ReceivePaymentResponse) {
	value.Destroy()
}

type RecommendedFees struct {
	FastestFee  uint64
	HalfHourFee uint64
	HourFee     uint64
	EconomyFee  uint64
	MinimumFee  uint64
}

func (r *RecommendedFees) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.FastestFee)
	FfiDestroyerUint64{}.Destroy(r.HalfHourFee)
	FfiDestroyerUint64{}.Destroy(r.HourFee)
	FfiDestroyerUint64{}.Destroy(r.EconomyFee)
	FfiDestroyerUint64{}.Destroy(r.MinimumFee)
}

type FfiConverterRecommendedFees struct{}

var FfiConverterRecommendedFeesINSTANCE = FfiConverterRecommendedFees{}

func (c FfiConverterRecommendedFees) Lift(rb RustBufferI) RecommendedFees {
	return LiftFromRustBuffer[RecommendedFees](c, rb)
}

func (c FfiConverterRecommendedFees) Read(reader io.Reader) RecommendedFees {
	return RecommendedFees{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRecommendedFees) Lower(value RecommendedFees) C.RustBuffer {
	return LowerIntoRustBuffer[RecommendedFees](c, value)
}

func (c FfiConverterRecommendedFees) Write(writer io.Writer, value RecommendedFees) {
	FfiConverterUint64INSTANCE.Write(writer, value.FastestFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HalfHourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.EconomyFee)
	FfiConverterUint64INSTANCE.Write(writer, value.MinimumFee)
}

type FfiDestroyerRecommendedFees struct{}

func (_ FfiDestroyerRecommendedFees) Destroy(value RecommendedFees) {
	value.Destroy()
}

type RefundRequest struct {
	SwapAddress        string
	RefundAddress      string
	FeeRateSatPerVbyte uint32
}

func (r *RefundRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapAddress)
	FfiDestroyerString{}.Destroy(r.RefundAddress)
	FfiDestroyerUint32{}.Destroy(r.FeeRateSatPerVbyte)
}

type FfiConverterRefundRequest struct{}

var FfiConverterRefundRequestINSTANCE = FfiConverterRefundRequest{}

func (c FfiConverterRefundRequest) Lift(rb RustBufferI) RefundRequest {
	return LiftFromRustBuffer[RefundRequest](c, rb)
}

func (c FfiConverterRefundRequest) Read(reader io.Reader) RefundRequest {
	return RefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterRefundRequest) Lower(value RefundRequest) C.RustBuffer {
	return LowerIntoRustBuffer[RefundRequest](c, value)
}

func (c FfiConverterRefundRequest) Write(writer io.Writer, value RefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.RefundAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerRefundRequest struct{}

func (_ FfiDestroyerRefundRequest) Destroy(value RefundRequest) {
	value.Destroy()
}

type RefundResponse struct {
	RefundTxId string
}

func (r *RefundResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.RefundTxId)
}

type FfiConverterRefundResponse struct{}

var FfiConverterRefundResponseINSTANCE = FfiConverterRefundResponse{}

func (c FfiConverterRefundResponse) Lift(rb RustBufferI) RefundResponse {
	return LiftFromRustBuffer[RefundResponse](c, rb)
}

func (c FfiConverterRefundResponse) Read(reader io.Reader) RefundResponse {
	return RefundResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterRefundResponse) Lower(value RefundResponse) C.RustBuffer {
	return LowerIntoRustBuffer[RefundResponse](c, value)
}

func (c FfiConverterRefundResponse) Write(writer io.Writer, value RefundResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.RefundTxId)
}

type FfiDestroyerRefundResponse struct{}

func (_ FfiDestroyerRefundResponse) Destroy(value RefundResponse) {
	value.Destroy()
}

type RefundableSwap struct {
	SwapAddress    string
	Timestamp      uint32
	AmountSat      uint64
	LastRefundTxId *string
}

func (r *RefundableSwap) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapAddress)
	FfiDestroyerUint32{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.LastRefundTxId)
}

type FfiConverterRefundableSwap struct{}

var FfiConverterRefundableSwapINSTANCE = FfiConverterRefundableSwap{}

func (c FfiConverterRefundableSwap) Lift(rb RustBufferI) RefundableSwap {
	return LiftFromRustBuffer[RefundableSwap](c, rb)
}

func (c FfiConverterRefundableSwap) Read(reader io.Reader) RefundableSwap {
	return RefundableSwap{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterRefundableSwap) Lower(value RefundableSwap) C.RustBuffer {
	return LowerIntoRustBuffer[RefundableSwap](c, value)
}

func (c FfiConverterRefundableSwap) Write(writer io.Writer, value RefundableSwap) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRefundTxId)
}

type FfiDestroyerRefundableSwap struct{}

func (_ FfiDestroyerRefundableSwap) Destroy(value RefundableSwap) {
	value.Destroy()
}

type RestoreRequest struct {
	BackupPath *string
}

func (r *RestoreRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.BackupPath)
}

type FfiConverterRestoreRequest struct{}

var FfiConverterRestoreRequestINSTANCE = FfiConverterRestoreRequest{}

func (c FfiConverterRestoreRequest) Lift(rb RustBufferI) RestoreRequest {
	return LiftFromRustBuffer[RestoreRequest](c, rb)
}

func (c FfiConverterRestoreRequest) Read(reader io.Reader) RestoreRequest {
	return RestoreRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterRestoreRequest) Lower(value RestoreRequest) C.RustBuffer {
	return LowerIntoRustBuffer[RestoreRequest](c, value)
}

func (c FfiConverterRestoreRequest) Write(writer io.Writer, value RestoreRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BackupPath)
}

type FfiDestroyerRestoreRequest struct{}

func (_ FfiDestroyerRestoreRequest) Destroy(value RestoreRequest) {
	value.Destroy()
}

type RouteHint struct {
	Hops []RouteHintHop
}

func (r *RouteHint) Destroy() {
	FfiDestroyerSequenceRouteHintHop{}.Destroy(r.Hops)
}

type FfiConverterRouteHint struct{}

var FfiConverterRouteHintINSTANCE = FfiConverterRouteHint{}

func (c FfiConverterRouteHint) Lift(rb RustBufferI) RouteHint {
	return LiftFromRustBuffer[RouteHint](c, rb)
}

func (c FfiConverterRouteHint) Read(reader io.Reader) RouteHint {
	return RouteHint{
		FfiConverterSequenceRouteHintHopINSTANCE.Read(reader),
	}
}

func (c FfiConverterRouteHint) Lower(value RouteHint) C.RustBuffer {
	return LowerIntoRustBuffer[RouteHint](c, value)
}

func (c FfiConverterRouteHint) Write(writer io.Writer, value RouteHint) {
	FfiConverterSequenceRouteHintHopINSTANCE.Write(writer, value.Hops)
}

type FfiDestroyerRouteHint struct{}

func (_ FfiDestroyerRouteHint) Destroy(value RouteHint) {
	value.Destroy()
}

type RouteHintHop struct {
	SrcNodeId                  string
	ShortChannelId             string
	FeesBaseMsat               uint32
	FeesProportionalMillionths uint32
	CltvExpiryDelta            uint64
	HtlcMinimumMsat            *uint64
	HtlcMaximumMsat            *uint64
}

func (r *RouteHintHop) Destroy() {
	FfiDestroyerString{}.Destroy(r.SrcNodeId)
	FfiDestroyerString{}.Destroy(r.ShortChannelId)
	FfiDestroyerUint32{}.Destroy(r.FeesBaseMsat)
	FfiDestroyerUint32{}.Destroy(r.FeesProportionalMillionths)
	FfiDestroyerUint64{}.Destroy(r.CltvExpiryDelta)
	FfiDestroyerOptionalUint64{}.Destroy(r.HtlcMinimumMsat)
	FfiDestroyerOptionalUint64{}.Destroy(r.HtlcMaximumMsat)
}

type FfiConverterRouteHintHop struct{}

var FfiConverterRouteHintHopINSTANCE = FfiConverterRouteHintHop{}

func (c FfiConverterRouteHintHop) Lift(rb RustBufferI) RouteHintHop {
	return LiftFromRustBuffer[RouteHintHop](c, rb)
}

func (c FfiConverterRouteHintHop) Read(reader io.Reader) RouteHintHop {
	return RouteHintHop{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRouteHintHop) Lower(value RouteHintHop) C.RustBuffer {
	return LowerIntoRustBuffer[RouteHintHop](c, value)
}

func (c FfiConverterRouteHintHop) Write(writer io.Writer, value RouteHintHop) {
	FfiConverterStringINSTANCE.Write(writer, value.SrcNodeId)
	FfiConverterStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesBaseMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesProportionalMillionths)
	FfiConverterUint64INSTANCE.Write(writer, value.CltvExpiryDelta)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMinimumMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMaximumMsat)
}

type FfiDestroyerRouteHintHop struct{}

func (_ FfiDestroyerRouteHintHop) Destroy(value RouteHintHop) {
	value.Destroy()
}

type SendPaymentRequest struct {
	PrepareResponse PrepareSendResponse
	UseAssetFees    *bool
	PayerNote       *string
}

func (r *SendPaymentRequest) Destroy() {
	FfiDestroyerPrepareSendResponse{}.Destroy(r.PrepareResponse)
	FfiDestroyerOptionalBool{}.Destroy(r.UseAssetFees)
	FfiDestroyerOptionalString{}.Destroy(r.PayerNote)
}

type FfiConverterSendPaymentRequest struct{}

var FfiConverterSendPaymentRequestINSTANCE = FfiConverterSendPaymentRequest{}

func (c FfiConverterSendPaymentRequest) Lift(rb RustBufferI) SendPaymentRequest {
	return LiftFromRustBuffer[SendPaymentRequest](c, rb)
}

func (c FfiConverterSendPaymentRequest) Read(reader io.Reader) SendPaymentRequest {
	return SendPaymentRequest{
		FfiConverterPrepareSendResponseINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSendPaymentRequest) Lower(value SendPaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[SendPaymentRequest](c, value)
}

func (c FfiConverterSendPaymentRequest) Write(writer io.Writer, value SendPaymentRequest) {
	FfiConverterPrepareSendResponseINSTANCE.Write(writer, value.PrepareResponse)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.UseAssetFees)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PayerNote)
}

type FfiDestroyerSendPaymentRequest struct{}

func (_ FfiDestroyerSendPaymentRequest) Destroy(value SendPaymentRequest) {
	value.Destroy()
}

type SendPaymentResponse struct {
	Payment Payment
}

func (r *SendPaymentResponse) Destroy() {
	FfiDestroyerPayment{}.Destroy(r.Payment)
}

type FfiConverterSendPaymentResponse struct{}

var FfiConverterSendPaymentResponseINSTANCE = FfiConverterSendPaymentResponse{}

func (c FfiConverterSendPaymentResponse) Lift(rb RustBufferI) SendPaymentResponse {
	return LiftFromRustBuffer[SendPaymentResponse](c, rb)
}

func (c FfiConverterSendPaymentResponse) Read(reader io.Reader) SendPaymentResponse {
	return SendPaymentResponse{
		FfiConverterPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterSendPaymentResponse) Lower(value SendPaymentResponse) C.RustBuffer {
	return LowerIntoRustBuffer[SendPaymentResponse](c, value)
}

func (c FfiConverterSendPaymentResponse) Write(writer io.Writer, value SendPaymentResponse) {
	FfiConverterPaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerSendPaymentResponse struct{}

func (_ FfiDestroyerSendPaymentResponse) Destroy(value SendPaymentResponse) {
	value.Destroy()
}

type SignMessageRequest struct {
	Message string
}

func (r *SignMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterSignMessageRequest struct{}

var FfiConverterSignMessageRequestINSTANCE = FfiConverterSignMessageRequest{}

func (c FfiConverterSignMessageRequest) Lift(rb RustBufferI) SignMessageRequest {
	return LiftFromRustBuffer[SignMessageRequest](c, rb)
}

func (c FfiConverterSignMessageRequest) Read(reader io.Reader) SignMessageRequest {
	return SignMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSignMessageRequest) Lower(value SignMessageRequest) C.RustBuffer {
	return LowerIntoRustBuffer[SignMessageRequest](c, value)
}

func (c FfiConverterSignMessageRequest) Write(writer io.Writer, value SignMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerSignMessageRequest struct{}

func (_ FfiDestroyerSignMessageRequest) Destroy(value SignMessageRequest) {
	value.Destroy()
}

type SignMessageResponse struct {
	Signature string
}

func (r *SignMessageResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterSignMessageResponse struct{}

var FfiConverterSignMessageResponseINSTANCE = FfiConverterSignMessageResponse{}

func (c FfiConverterSignMessageResponse) Lift(rb RustBufferI) SignMessageResponse {
	return LiftFromRustBuffer[SignMessageResponse](c, rb)
}

func (c FfiConverterSignMessageResponse) Read(reader io.Reader) SignMessageResponse {
	return SignMessageResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSignMessageResponse) Lower(value SignMessageResponse) C.RustBuffer {
	return LowerIntoRustBuffer[SignMessageResponse](c, value)
}

func (c FfiConverterSignMessageResponse) Write(writer io.Writer, value SignMessageResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerSignMessageResponse struct{}

func (_ FfiDestroyerSignMessageResponse) Destroy(value SignMessageResponse) {
	value.Destroy()
}

type Symbol struct {
	Grapheme *string
	Template *string
	Rtl      *bool
	Position *uint32
}

func (r *Symbol) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.Grapheme)
	FfiDestroyerOptionalString{}.Destroy(r.Template)
	FfiDestroyerOptionalBool{}.Destroy(r.Rtl)
	FfiDestroyerOptionalUint32{}.Destroy(r.Position)
}

type FfiConverterSymbol struct{}

var FfiConverterSymbolINSTANCE = FfiConverterSymbol{}

func (c FfiConverterSymbol) Lift(rb RustBufferI) Symbol {
	return LiftFromRustBuffer[Symbol](c, rb)
}

func (c FfiConverterSymbol) Read(reader io.Reader) Symbol {
	return Symbol{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterSymbol) Lower(value Symbol) C.RustBuffer {
	return LowerIntoRustBuffer[Symbol](c, value)
}

func (c FfiConverterSymbol) Write(writer io.Writer, value Symbol) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Grapheme)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Template)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Rtl)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Position)
}

type FfiDestroyerSymbol struct{}

func (_ FfiDestroyerSymbol) Destroy(value Symbol) {
	value.Destroy()
}

type UrlSuccessActionData struct {
	Description           string
	Url                   string
	MatchesCallbackDomain bool
}

func (r *UrlSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerBool{}.Destroy(r.MatchesCallbackDomain)
}

type FfiConverterUrlSuccessActionData struct{}

var FfiConverterUrlSuccessActionDataINSTANCE = FfiConverterUrlSuccessActionData{}

func (c FfiConverterUrlSuccessActionData) Lift(rb RustBufferI) UrlSuccessActionData {
	return LiftFromRustBuffer[UrlSuccessActionData](c, rb)
}

func (c FfiConverterUrlSuccessActionData) Read(reader io.Reader) UrlSuccessActionData {
	return UrlSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterUrlSuccessActionData) Lower(value UrlSuccessActionData) C.RustBuffer {
	return LowerIntoRustBuffer[UrlSuccessActionData](c, value)
}

func (c FfiConverterUrlSuccessActionData) Write(writer io.Writer, value UrlSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterBoolINSTANCE.Write(writer, value.MatchesCallbackDomain)
}

type FfiDestroyerUrlSuccessActionData struct{}

func (_ FfiDestroyerUrlSuccessActionData) Destroy(value UrlSuccessActionData) {
	value.Destroy()
}

type WalletInfo struct {
	BalanceSat        uint64
	PendingSendSat    uint64
	PendingReceiveSat uint64
	Fingerprint       string
	Pubkey            string
	AssetBalances     []AssetBalance
}

func (r *WalletInfo) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.BalanceSat)
	FfiDestroyerUint64{}.Destroy(r.PendingSendSat)
	FfiDestroyerUint64{}.Destroy(r.PendingReceiveSat)
	FfiDestroyerString{}.Destroy(r.Fingerprint)
	FfiDestroyerString{}.Destroy(r.Pubkey)
	FfiDestroyerSequenceAssetBalance{}.Destroy(r.AssetBalances)
}

type FfiConverterWalletInfo struct{}

var FfiConverterWalletInfoINSTANCE = FfiConverterWalletInfo{}

func (c FfiConverterWalletInfo) Lift(rb RustBufferI) WalletInfo {
	return LiftFromRustBuffer[WalletInfo](c, rb)
}

func (c FfiConverterWalletInfo) Read(reader io.Reader) WalletInfo {
	return WalletInfo{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterSequenceAssetBalanceINSTANCE.Read(reader),
	}
}

func (c FfiConverterWalletInfo) Lower(value WalletInfo) C.RustBuffer {
	return LowerIntoRustBuffer[WalletInfo](c, value)
}

func (c FfiConverterWalletInfo) Write(writer io.Writer, value WalletInfo) {
	FfiConverterUint64INSTANCE.Write(writer, value.BalanceSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingSendSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingReceiveSat)
	FfiConverterStringINSTANCE.Write(writer, value.Fingerprint)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterSequenceAssetBalanceINSTANCE.Write(writer, value.AssetBalances)
}

type FfiDestroyerWalletInfo struct{}

func (_ FfiDestroyerWalletInfo) Destroy(value WalletInfo) {
	value.Destroy()
}

type AesSuccessActionDataResult interface {
	Destroy()
}
type AesSuccessActionDataResultDecrypted struct {
	Data AesSuccessActionDataDecrypted
}

func (e AesSuccessActionDataResultDecrypted) Destroy() {
	FfiDestroyerAesSuccessActionDataDecrypted{}.Destroy(e.Data)
}

type AesSuccessActionDataResultErrorStatus struct {
	Reason string
}

func (e AesSuccessActionDataResultErrorStatus) Destroy() {
	FfiDestroyerString{}.Destroy(e.Reason)
}

type FfiConverterAesSuccessActionDataResult struct{}

var FfiConverterAesSuccessActionDataResultINSTANCE = FfiConverterAesSuccessActionDataResult{}

func (c FfiConverterAesSuccessActionDataResult) Lift(rb RustBufferI) AesSuccessActionDataResult {
	return LiftFromRustBuffer[AesSuccessActionDataResult](c, rb)
}

func (c FfiConverterAesSuccessActionDataResult) Lower(value AesSuccessActionDataResult) C.RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataResult](c, value)
}
func (FfiConverterAesSuccessActionDataResult) Read(reader io.Reader) AesSuccessActionDataResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return AesSuccessActionDataResultDecrypted{
			FfiConverterAesSuccessActionDataDecryptedINSTANCE.Read(reader),
		}
	case 2:
		return AesSuccessActionDataResultErrorStatus{
			FfiConverterStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterAesSuccessActionDataResult.Read()", id))
	}
}

func (FfiConverterAesSuccessActionDataResult) Write(writer io.Writer, value AesSuccessActionDataResult) {
	switch variant_value := value.(type) {
	case AesSuccessActionDataResultDecrypted:
		writeInt32(writer, 1)
		FfiConverterAesSuccessActionDataDecryptedINSTANCE.Write(writer, variant_value.Data)
	case AesSuccessActionDataResultErrorStatus:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Reason)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterAesSuccessActionDataResult.Write", value))
	}
}

type FfiDestroyerAesSuccessActionDataResult struct{}

func (_ FfiDestroyerAesSuccessActionDataResult) Destroy(value AesSuccessActionDataResult) {
	value.Destroy()
}

type Amount interface {
	Destroy()
}
type AmountBitcoin struct {
	AmountMsat uint64
}

func (e AmountBitcoin) Destroy() {
	FfiDestroyerUint64{}.Destroy(e.AmountMsat)
}

type AmountCurrency struct {
	Iso4217Code      string
	FractionalAmount uint64
}

func (e AmountCurrency) Destroy() {
	FfiDestroyerString{}.Destroy(e.Iso4217Code)
	FfiDestroyerUint64{}.Destroy(e.FractionalAmount)
}

type FfiConverterAmount struct{}

var FfiConverterAmountINSTANCE = FfiConverterAmount{}

func (c FfiConverterAmount) Lift(rb RustBufferI) Amount {
	return LiftFromRustBuffer[Amount](c, rb)
}

func (c FfiConverterAmount) Lower(value Amount) C.RustBuffer {
	return LowerIntoRustBuffer[Amount](c, value)
}
func (FfiConverterAmount) Read(reader io.Reader) Amount {
	id := readInt32(reader)
	switch id {
	case 1:
		return AmountBitcoin{
			FfiConverterUint64INSTANCE.Read(reader),
		}
	case 2:
		return AmountCurrency{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterUint64INSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterAmount.Read()", id))
	}
}

func (FfiConverterAmount) Write(writer io.Writer, value Amount) {
	switch variant_value := value.(type) {
	case AmountBitcoin:
		writeInt32(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.AmountMsat)
	case AmountCurrency:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Iso4217Code)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.FractionalAmount)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterAmount.Write", value))
	}
}

type FfiDestroyerAmount struct{}

func (_ FfiDestroyerAmount) Destroy(value Amount) {
	value.Destroy()
}

type BlockchainExplorer interface {
	Destroy()
}
type BlockchainExplorerElectrum struct {
	Url string
}

func (e BlockchainExplorerElectrum) Destroy() {
	FfiDestroyerString{}.Destroy(e.Url)
}

type BlockchainExplorerEsplora struct {
	Url           string
	UseWaterfalls bool
}

func (e BlockchainExplorerEsplora) Destroy() {
	FfiDestroyerString{}.Destroy(e.Url)
	FfiDestroyerBool{}.Destroy(e.UseWaterfalls)
}

type FfiConverterBlockchainExplorer struct{}

var FfiConverterBlockchainExplorerINSTANCE = FfiConverterBlockchainExplorer{}

func (c FfiConverterBlockchainExplorer) Lift(rb RustBufferI) BlockchainExplorer {
	return LiftFromRustBuffer[BlockchainExplorer](c, rb)
}

func (c FfiConverterBlockchainExplorer) Lower(value BlockchainExplorer) C.RustBuffer {
	return LowerIntoRustBuffer[BlockchainExplorer](c, value)
}
func (FfiConverterBlockchainExplorer) Read(reader io.Reader) BlockchainExplorer {
	id := readInt32(reader)
	switch id {
	case 1:
		return BlockchainExplorerElectrum{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 2:
		return BlockchainExplorerEsplora{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterBoolINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterBlockchainExplorer.Read()", id))
	}
}

func (FfiConverterBlockchainExplorer) Write(writer io.Writer, value BlockchainExplorer) {
	switch variant_value := value.(type) {
	case BlockchainExplorerElectrum:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
	case BlockchainExplorerEsplora:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
		FfiConverterBoolINSTANCE.Write(writer, variant_value.UseWaterfalls)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterBlockchainExplorer.Write", value))
	}
}

type FfiDestroyerBlockchainExplorer struct{}

func (_ FfiDestroyerBlockchainExplorer) Destroy(value BlockchainExplorer) {
	value.Destroy()
}

type BuyBitcoinProvider uint

const (
	BuyBitcoinProviderMoonpay BuyBitcoinProvider = 1
)

type FfiConverterBuyBitcoinProvider struct{}

var FfiConverterBuyBitcoinProviderINSTANCE = FfiConverterBuyBitcoinProvider{}

func (c FfiConverterBuyBitcoinProvider) Lift(rb RustBufferI) BuyBitcoinProvider {
	return LiftFromRustBuffer[BuyBitcoinProvider](c, rb)
}

func (c FfiConverterBuyBitcoinProvider) Lower(value BuyBitcoinProvider) C.RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinProvider](c, value)
}
func (FfiConverterBuyBitcoinProvider) Read(reader io.Reader) BuyBitcoinProvider {
	id := readInt32(reader)
	return BuyBitcoinProvider(id)
}

func (FfiConverterBuyBitcoinProvider) Write(writer io.Writer, value BuyBitcoinProvider) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerBuyBitcoinProvider struct{}

func (_ FfiDestroyerBuyBitcoinProvider) Destroy(value BuyBitcoinProvider) {
}

type GetPaymentRequest interface {
	Destroy()
}
type GetPaymentRequestPaymentHash struct {
	PaymentHash string
}

func (e GetPaymentRequestPaymentHash) Destroy() {
	FfiDestroyerString{}.Destroy(e.PaymentHash)
}

type GetPaymentRequestSwapId struct {
	SwapId string
}

func (e GetPaymentRequestSwapId) Destroy() {
	FfiDestroyerString{}.Destroy(e.SwapId)
}

type FfiConverterGetPaymentRequest struct{}

var FfiConverterGetPaymentRequestINSTANCE = FfiConverterGetPaymentRequest{}

func (c FfiConverterGetPaymentRequest) Lift(rb RustBufferI) GetPaymentRequest {
	return LiftFromRustBuffer[GetPaymentRequest](c, rb)
}

func (c FfiConverterGetPaymentRequest) Lower(value GetPaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[GetPaymentRequest](c, value)
}
func (FfiConverterGetPaymentRequest) Read(reader io.Reader) GetPaymentRequest {
	id := readInt32(reader)
	switch id {
	case 1:
		return GetPaymentRequestPaymentHash{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 2:
		return GetPaymentRequestSwapId{
			FfiConverterStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterGetPaymentRequest.Read()", id))
	}
}

func (FfiConverterGetPaymentRequest) Write(writer io.Writer, value GetPaymentRequest) {
	switch variant_value := value.(type) {
	case GetPaymentRequestPaymentHash:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variant_value.PaymentHash)
	case GetPaymentRequestSwapId:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.SwapId)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterGetPaymentRequest.Write", value))
	}
}

type FfiDestroyerGetPaymentRequest struct{}

func (_ FfiDestroyerGetPaymentRequest) Destroy(value GetPaymentRequest) {
	value.Destroy()
}

type InputType interface {
	Destroy()
}
type InputTypeBitcoinAddress struct {
	Address BitcoinAddressData
}

func (e InputTypeBitcoinAddress) Destroy() {
	FfiDestroyerBitcoinAddressData{}.Destroy(e.Address)
}

type InputTypeLiquidAddress struct {
	Address LiquidAddressData
}

func (e InputTypeLiquidAddress) Destroy() {
	FfiDestroyerLiquidAddressData{}.Destroy(e.Address)
}

type InputTypeBolt11 struct {
	Invoice LnInvoice
}

func (e InputTypeBolt11) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(e.Invoice)
}

type InputTypeBolt12Offer struct {
	Offer         LnOffer
	Bip353Address *string
}

func (e InputTypeBolt12Offer) Destroy() {
	FfiDestroyerLnOffer{}.Destroy(e.Offer)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type InputTypeNodeId struct {
	NodeId string
}

func (e InputTypeNodeId) Destroy() {
	FfiDestroyerString{}.Destroy(e.NodeId)
}

type InputTypeUrl struct {
	Url string
}

func (e InputTypeUrl) Destroy() {
	FfiDestroyerString{}.Destroy(e.Url)
}

type InputTypeLnUrlPay struct {
	Data          LnUrlPayRequestData
	Bip353Address *string
}

func (e InputTypeLnUrlPay) Destroy() {
	FfiDestroyerLnUrlPayRequestData{}.Destroy(e.Data)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type InputTypeLnUrlWithdraw struct {
	Data LnUrlWithdrawRequestData
}

func (e InputTypeLnUrlWithdraw) Destroy() {
	FfiDestroyerLnUrlWithdrawRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlAuth struct {
	Data LnUrlAuthRequestData
}

func (e InputTypeLnUrlAuth) Destroy() {
	FfiDestroyerLnUrlAuthRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlError struct {
	Data LnUrlErrorData
}

func (e InputTypeLnUrlError) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterInputType struct{}

var FfiConverterInputTypeINSTANCE = FfiConverterInputType{}

func (c FfiConverterInputType) Lift(rb RustBufferI) InputType {
	return LiftFromRustBuffer[InputType](c, rb)
}

func (c FfiConverterInputType) Lower(value InputType) C.RustBuffer {
	return LowerIntoRustBuffer[InputType](c, value)
}
func (FfiConverterInputType) Read(reader io.Reader) InputType {
	id := readInt32(reader)
	switch id {
	case 1:
		return InputTypeBitcoinAddress{
			FfiConverterBitcoinAddressDataINSTANCE.Read(reader),
		}
	case 2:
		return InputTypeLiquidAddress{
			FfiConverterLiquidAddressDataINSTANCE.Read(reader),
		}
	case 3:
		return InputTypeBolt11{
			FfiConverterLnInvoiceINSTANCE.Read(reader),
		}
	case 4:
		return InputTypeBolt12Offer{
			FfiConverterLnOfferINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 5:
		return InputTypeNodeId{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 6:
		return InputTypeUrl{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 7:
		return InputTypeLnUrlPay{
			FfiConverterLnUrlPayRequestDataINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 8:
		return InputTypeLnUrlWithdraw{
			FfiConverterLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		}
	case 9:
		return InputTypeLnUrlAuth{
			FfiConverterLnUrlAuthRequestDataINSTANCE.Read(reader),
		}
	case 10:
		return InputTypeLnUrlError{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterInputType.Read()", id))
	}
}

func (FfiConverterInputType) Write(writer io.Writer, value InputType) {
	switch variant_value := value.(type) {
	case InputTypeBitcoinAddress:
		writeInt32(writer, 1)
		FfiConverterBitcoinAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeLiquidAddress:
		writeInt32(writer, 2)
		FfiConverterLiquidAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeBolt11:
		writeInt32(writer, 3)
		FfiConverterLnInvoiceINSTANCE.Write(writer, variant_value.Invoice)
	case InputTypeBolt12Offer:
		writeInt32(writer, 4)
		FfiConverterLnOfferINSTANCE.Write(writer, variant_value.Offer)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case InputTypeNodeId:
		writeInt32(writer, 5)
		FfiConverterStringINSTANCE.Write(writer, variant_value.NodeId)
	case InputTypeUrl:
		writeInt32(writer, 6)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
	case InputTypeLnUrlPay:
		writeInt32(writer, 7)
		FfiConverterLnUrlPayRequestDataINSTANCE.Write(writer, variant_value.Data)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case InputTypeLnUrlWithdraw:
		writeInt32(writer, 8)
		FfiConverterLnUrlWithdrawRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlAuth:
		writeInt32(writer, 9)
		FfiConverterLnUrlAuthRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlError:
		writeInt32(writer, 10)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterInputType.Write", value))
	}
}

type FfiDestroyerInputType struct{}

func (_ FfiDestroyerInputType) Destroy(value InputType) {
	value.Destroy()
}

type LiquidNetwork uint

const (
	LiquidNetworkMainnet LiquidNetwork = 1
	LiquidNetworkTestnet LiquidNetwork = 2
	LiquidNetworkRegtest LiquidNetwork = 3
)

type FfiConverterLiquidNetwork struct{}

var FfiConverterLiquidNetworkINSTANCE = FfiConverterLiquidNetwork{}

func (c FfiConverterLiquidNetwork) Lift(rb RustBufferI) LiquidNetwork {
	return LiftFromRustBuffer[LiquidNetwork](c, rb)
}

func (c FfiConverterLiquidNetwork) Lower(value LiquidNetwork) C.RustBuffer {
	return LowerIntoRustBuffer[LiquidNetwork](c, value)
}
func (FfiConverterLiquidNetwork) Read(reader io.Reader) LiquidNetwork {
	id := readInt32(reader)
	return LiquidNetwork(id)
}

func (FfiConverterLiquidNetwork) Write(writer io.Writer, value LiquidNetwork) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerLiquidNetwork struct{}

func (_ FfiDestroyerLiquidNetwork) Destroy(value LiquidNetwork) {
}

type ListPaymentDetails interface {
	Destroy()
}
type ListPaymentDetailsLiquid struct {
	AssetId     *string
	Destination *string
}

func (e ListPaymentDetailsLiquid) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(e.AssetId)
	FfiDestroyerOptionalString{}.Destroy(e.Destination)
}

type ListPaymentDetailsBitcoin struct {
	Address *string
}

func (e ListPaymentDetailsBitcoin) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(e.Address)
}

type FfiConverterListPaymentDetails struct{}

var FfiConverterListPaymentDetailsINSTANCE = FfiConverterListPaymentDetails{}

func (c FfiConverterListPaymentDetails) Lift(rb RustBufferI) ListPaymentDetails {
	return LiftFromRustBuffer[ListPaymentDetails](c, rb)
}

func (c FfiConverterListPaymentDetails) Lower(value ListPaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[ListPaymentDetails](c, value)
}
func (FfiConverterListPaymentDetails) Read(reader io.Reader) ListPaymentDetails {
	id := readInt32(reader)
	switch id {
	case 1:
		return ListPaymentDetailsLiquid{
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 2:
		return ListPaymentDetailsBitcoin{
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterListPaymentDetails.Read()", id))
	}
}

func (FfiConverterListPaymentDetails) Write(writer io.Writer, value ListPaymentDetails) {
	switch variant_value := value.(type) {
	case ListPaymentDetailsLiquid:
		writeInt32(writer, 1)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Destination)
	case ListPaymentDetailsBitcoin:
		writeInt32(writer, 2)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Address)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterListPaymentDetails.Write", value))
	}
}

type FfiDestroyerListPaymentDetails struct{}

func (_ FfiDestroyerListPaymentDetails) Destroy(value ListPaymentDetails) {
	value.Destroy()
}

type LnUrlAuthError struct {
	err error
}

// Convience method to turn *LnUrlAuthError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlAuthError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlAuthError) Error() string {
	return fmt.Sprintf("LnUrlAuthError: %s", err.err.Error())
}

func (err LnUrlAuthError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlAuthErrorGeneric = fmt.Errorf("LnUrlAuthErrorGeneric")
var ErrLnUrlAuthErrorInvalidUri = fmt.Errorf("LnUrlAuthErrorInvalidUri")
var ErrLnUrlAuthErrorServiceConnectivity = fmt.Errorf("LnUrlAuthErrorServiceConnectivity")

// Variant structs
type LnUrlAuthErrorGeneric struct {
	Err string
}

func NewLnUrlAuthErrorGeneric(
	err string,
) *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorGeneric{
		Err: err}}
}

func (e LnUrlAuthErrorGeneric) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlAuthErrorGeneric) Error() string {
	return fmt.Sprint("Generic",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlAuthErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlAuthErrorGeneric
}

type LnUrlAuthErrorInvalidUri struct {
	Err string
}

func NewLnUrlAuthErrorInvalidUri(
	err string,
) *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorInvalidUri{
		Err: err}}
}

func (e LnUrlAuthErrorInvalidUri) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlAuthErrorInvalidUri) Error() string {
	return fmt.Sprint("InvalidUri",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlAuthErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlAuthErrorInvalidUri
}

type LnUrlAuthErrorServiceConnectivity struct {
	Err string
}

func NewLnUrlAuthErrorServiceConnectivity(
	err string,
) *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorServiceConnectivity{
		Err: err}}
}

func (e LnUrlAuthErrorServiceConnectivity) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlAuthErrorServiceConnectivity) Error() string {
	return fmt.Sprint("ServiceConnectivity",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlAuthErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlAuthErrorServiceConnectivity
}

type FfiConverterLnUrlAuthError struct{}

var FfiConverterLnUrlAuthErrorINSTANCE = FfiConverterLnUrlAuthError{}

func (c FfiConverterLnUrlAuthError) Lift(eb RustBufferI) *LnUrlAuthError {
	return LiftFromRustBuffer[*LnUrlAuthError](c, eb)
}

func (c FfiConverterLnUrlAuthError) Lower(value *LnUrlAuthError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlAuthError](c, value)
}

func (c FfiConverterLnUrlAuthError) Read(reader io.Reader) *LnUrlAuthError {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &LnUrlAuthError{&LnUrlAuthErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 2:
		return &LnUrlAuthError{&LnUrlAuthErrorInvalidUri{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 3:
		return &LnUrlAuthError{&LnUrlAuthErrorServiceConnectivity{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlAuthError.Read()", errorID))
	}
}

func (c FfiConverterLnUrlAuthError) Write(writer io.Writer, value *LnUrlAuthError) {
	switch variantValue := value.err.(type) {
	case *LnUrlAuthErrorGeneric:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlAuthErrorInvalidUri:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlAuthErrorServiceConnectivity:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlAuthError.Write", value))
	}
}

type FfiDestroyerLnUrlAuthError struct{}

func (_ FfiDestroyerLnUrlAuthError) Destroy(value *LnUrlAuthError) {
	switch variantValue := value.err.(type) {
	case LnUrlAuthErrorGeneric:
		variantValue.destroy()
	case LnUrlAuthErrorInvalidUri:
		variantValue.destroy()
	case LnUrlAuthErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlAuthError.Destroy", value))
	}
}

type LnUrlCallbackStatus interface {
	Destroy()
}
type LnUrlCallbackStatusOk struct {
}

func (e LnUrlCallbackStatusOk) Destroy() {
}

type LnUrlCallbackStatusErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlCallbackStatusErrorStatus) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlCallbackStatus struct{}

var FfiConverterLnUrlCallbackStatusINSTANCE = FfiConverterLnUrlCallbackStatus{}

func (c FfiConverterLnUrlCallbackStatus) Lift(rb RustBufferI) LnUrlCallbackStatus {
	return LiftFromRustBuffer[LnUrlCallbackStatus](c, rb)
}

func (c FfiConverterLnUrlCallbackStatus) Lower(value LnUrlCallbackStatus) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlCallbackStatus](c, value)
}
func (FfiConverterLnUrlCallbackStatus) Read(reader io.Reader) LnUrlCallbackStatus {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlCallbackStatusOk{}
	case 2:
		return LnUrlCallbackStatusErrorStatus{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlCallbackStatus.Read()", id))
	}
}

func (FfiConverterLnUrlCallbackStatus) Write(writer io.Writer, value LnUrlCallbackStatus) {
	switch variant_value := value.(type) {
	case LnUrlCallbackStatusOk:
		writeInt32(writer, 1)
	case LnUrlCallbackStatusErrorStatus:
		writeInt32(writer, 2)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlCallbackStatus.Write", value))
	}
}

type FfiDestroyerLnUrlCallbackStatus struct{}

func (_ FfiDestroyerLnUrlCallbackStatus) Destroy(value LnUrlCallbackStatus) {
	value.Destroy()
}

type LnUrlPayError struct {
	err error
}

// Convience method to turn *LnUrlPayError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlPayError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlPayError) Error() string {
	return fmt.Sprintf("LnUrlPayError: %s", err.err.Error())
}

func (err LnUrlPayError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlPayErrorAlreadyPaid = fmt.Errorf("LnUrlPayErrorAlreadyPaid")
var ErrLnUrlPayErrorGeneric = fmt.Errorf("LnUrlPayErrorGeneric")
var ErrLnUrlPayErrorInsufficientBalance = fmt.Errorf("LnUrlPayErrorInsufficientBalance")
var ErrLnUrlPayErrorInvalidAmount = fmt.Errorf("LnUrlPayErrorInvalidAmount")
var ErrLnUrlPayErrorInvalidInvoice = fmt.Errorf("LnUrlPayErrorInvalidInvoice")
var ErrLnUrlPayErrorInvalidNetwork = fmt.Errorf("LnUrlPayErrorInvalidNetwork")
var ErrLnUrlPayErrorInvalidUri = fmt.Errorf("LnUrlPayErrorInvalidUri")
var ErrLnUrlPayErrorInvoiceExpired = fmt.Errorf("LnUrlPayErrorInvoiceExpired")
var ErrLnUrlPayErrorPaymentFailed = fmt.Errorf("LnUrlPayErrorPaymentFailed")
var ErrLnUrlPayErrorPaymentTimeout = fmt.Errorf("LnUrlPayErrorPaymentTimeout")
var ErrLnUrlPayErrorRouteNotFound = fmt.Errorf("LnUrlPayErrorRouteNotFound")
var ErrLnUrlPayErrorRouteTooExpensive = fmt.Errorf("LnUrlPayErrorRouteTooExpensive")
var ErrLnUrlPayErrorServiceConnectivity = fmt.Errorf("LnUrlPayErrorServiceConnectivity")

// Variant structs
type LnUrlPayErrorAlreadyPaid struct {
}

func NewLnUrlPayErrorAlreadyPaid() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorAlreadyPaid{}}
}

func (e LnUrlPayErrorAlreadyPaid) destroy() {
}

func (err LnUrlPayErrorAlreadyPaid) Error() string {
	return fmt.Sprint("AlreadyPaid")
}

func (self LnUrlPayErrorAlreadyPaid) Is(target error) bool {
	return target == ErrLnUrlPayErrorAlreadyPaid
}

type LnUrlPayErrorGeneric struct {
	Err string
}

func NewLnUrlPayErrorGeneric(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorGeneric{
		Err: err}}
}

func (e LnUrlPayErrorGeneric) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorGeneric) Error() string {
	return fmt.Sprint("Generic",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlPayErrorGeneric
}

type LnUrlPayErrorInsufficientBalance struct {
	Err string
}

func NewLnUrlPayErrorInsufficientBalance(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInsufficientBalance{
		Err: err}}
}

func (e LnUrlPayErrorInsufficientBalance) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInsufficientBalance) Error() string {
	return fmt.Sprint("InsufficientBalance",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInsufficientBalance) Is(target error) bool {
	return target == ErrLnUrlPayErrorInsufficientBalance
}

type LnUrlPayErrorInvalidAmount struct {
	Err string
}

func NewLnUrlPayErrorInvalidAmount(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidAmount{
		Err: err}}
}

func (e LnUrlPayErrorInvalidAmount) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInvalidAmount) Error() string {
	return fmt.Sprint("InvalidAmount",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInvalidAmount) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidAmount
}

type LnUrlPayErrorInvalidInvoice struct {
	Err string
}

func NewLnUrlPayErrorInvalidInvoice(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidInvoice{
		Err: err}}
}

func (e LnUrlPayErrorInvalidInvoice) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInvalidInvoice) Error() string {
	return fmt.Sprint("InvalidInvoice",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInvalidInvoice) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidInvoice
}

type LnUrlPayErrorInvalidNetwork struct {
	Err string
}

func NewLnUrlPayErrorInvalidNetwork(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidNetwork{
		Err: err}}
}

func (e LnUrlPayErrorInvalidNetwork) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInvalidNetwork) Error() string {
	return fmt.Sprint("InvalidNetwork",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInvalidNetwork) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidNetwork
}

type LnUrlPayErrorInvalidUri struct {
	Err string
}

func NewLnUrlPayErrorInvalidUri(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidUri{
		Err: err}}
}

func (e LnUrlPayErrorInvalidUri) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInvalidUri) Error() string {
	return fmt.Sprint("InvalidUri",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidUri
}

type LnUrlPayErrorInvoiceExpired struct {
	Err string
}

func NewLnUrlPayErrorInvoiceExpired(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvoiceExpired{
		Err: err}}
}

func (e LnUrlPayErrorInvoiceExpired) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorInvoiceExpired) Error() string {
	return fmt.Sprint("InvoiceExpired",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorInvoiceExpired) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvoiceExpired
}

type LnUrlPayErrorPaymentFailed struct {
	Err string
}

func NewLnUrlPayErrorPaymentFailed(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorPaymentFailed{
		Err: err}}
}

func (e LnUrlPayErrorPaymentFailed) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorPaymentFailed) Error() string {
	return fmt.Sprint("PaymentFailed",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorPaymentFailed) Is(target error) bool {
	return target == ErrLnUrlPayErrorPaymentFailed
}

type LnUrlPayErrorPaymentTimeout struct {
	Err string
}

func NewLnUrlPayErrorPaymentTimeout(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorPaymentTimeout{
		Err: err}}
}

func (e LnUrlPayErrorPaymentTimeout) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorPaymentTimeout) Error() string {
	return fmt.Sprint("PaymentTimeout",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorPaymentTimeout) Is(target error) bool {
	return target == ErrLnUrlPayErrorPaymentTimeout
}

type LnUrlPayErrorRouteNotFound struct {
	Err string
}

func NewLnUrlPayErrorRouteNotFound(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorRouteNotFound{
		Err: err}}
}

func (e LnUrlPayErrorRouteNotFound) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorRouteNotFound) Error() string {
	return fmt.Sprint("RouteNotFound",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorRouteNotFound) Is(target error) bool {
	return target == ErrLnUrlPayErrorRouteNotFound
}

type LnUrlPayErrorRouteTooExpensive struct {
	Err string
}

func NewLnUrlPayErrorRouteTooExpensive(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorRouteTooExpensive{
		Err: err}}
}

func (e LnUrlPayErrorRouteTooExpensive) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorRouteTooExpensive) Error() string {
	return fmt.Sprint("RouteTooExpensive",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorRouteTooExpensive) Is(target error) bool {
	return target == ErrLnUrlPayErrorRouteTooExpensive
}

type LnUrlPayErrorServiceConnectivity struct {
	Err string
}

func NewLnUrlPayErrorServiceConnectivity(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorServiceConnectivity{
		Err: err}}
}

func (e LnUrlPayErrorServiceConnectivity) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlPayErrorServiceConnectivity) Error() string {
	return fmt.Sprint("ServiceConnectivity",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlPayErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlPayErrorServiceConnectivity
}

type FfiConverterLnUrlPayError struct{}

var FfiConverterLnUrlPayErrorINSTANCE = FfiConverterLnUrlPayError{}

func (c FfiConverterLnUrlPayError) Lift(eb RustBufferI) *LnUrlPayError {
	return LiftFromRustBuffer[*LnUrlPayError](c, eb)
}

func (c FfiConverterLnUrlPayError) Lower(value *LnUrlPayError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlPayError](c, value)
}

func (c FfiConverterLnUrlPayError) Read(reader io.Reader) *LnUrlPayError {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &LnUrlPayError{&LnUrlPayErrorAlreadyPaid{}}
	case 2:
		return &LnUrlPayError{&LnUrlPayErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 3:
		return &LnUrlPayError{&LnUrlPayErrorInsufficientBalance{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 4:
		return &LnUrlPayError{&LnUrlPayErrorInvalidAmount{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 5:
		return &LnUrlPayError{&LnUrlPayErrorInvalidInvoice{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 6:
		return &LnUrlPayError{&LnUrlPayErrorInvalidNetwork{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 7:
		return &LnUrlPayError{&LnUrlPayErrorInvalidUri{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 8:
		return &LnUrlPayError{&LnUrlPayErrorInvoiceExpired{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 9:
		return &LnUrlPayError{&LnUrlPayErrorPaymentFailed{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 10:
		return &LnUrlPayError{&LnUrlPayErrorPaymentTimeout{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 11:
		return &LnUrlPayError{&LnUrlPayErrorRouteNotFound{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 12:
		return &LnUrlPayError{&LnUrlPayErrorRouteTooExpensive{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 13:
		return &LnUrlPayError{&LnUrlPayErrorServiceConnectivity{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlPayError.Read()", errorID))
	}
}

func (c FfiConverterLnUrlPayError) Write(writer io.Writer, value *LnUrlPayError) {
	switch variantValue := value.err.(type) {
	case *LnUrlPayErrorAlreadyPaid:
		writeInt32(writer, 1)
	case *LnUrlPayErrorGeneric:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInsufficientBalance:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidAmount:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidInvoice:
		writeInt32(writer, 5)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidNetwork:
		writeInt32(writer, 6)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidUri:
		writeInt32(writer, 7)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvoiceExpired:
		writeInt32(writer, 8)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorPaymentFailed:
		writeInt32(writer, 9)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorPaymentTimeout:
		writeInt32(writer, 10)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorRouteNotFound:
		writeInt32(writer, 11)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorRouteTooExpensive:
		writeInt32(writer, 12)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorServiceConnectivity:
		writeInt32(writer, 13)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlPayError.Write", value))
	}
}

type FfiDestroyerLnUrlPayError struct{}

func (_ FfiDestroyerLnUrlPayError) Destroy(value *LnUrlPayError) {
	switch variantValue := value.err.(type) {
	case LnUrlPayErrorAlreadyPaid:
		variantValue.destroy()
	case LnUrlPayErrorGeneric:
		variantValue.destroy()
	case LnUrlPayErrorInsufficientBalance:
		variantValue.destroy()
	case LnUrlPayErrorInvalidAmount:
		variantValue.destroy()
	case LnUrlPayErrorInvalidInvoice:
		variantValue.destroy()
	case LnUrlPayErrorInvalidNetwork:
		variantValue.destroy()
	case LnUrlPayErrorInvalidUri:
		variantValue.destroy()
	case LnUrlPayErrorInvoiceExpired:
		variantValue.destroy()
	case LnUrlPayErrorPaymentFailed:
		variantValue.destroy()
	case LnUrlPayErrorPaymentTimeout:
		variantValue.destroy()
	case LnUrlPayErrorRouteNotFound:
		variantValue.destroy()
	case LnUrlPayErrorRouteTooExpensive:
		variantValue.destroy()
	case LnUrlPayErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlPayError.Destroy", value))
	}
}

// /////////////////////////////
// ///////////////////////////////
type LnUrlPayResult interface {
	Destroy()
}
type LnUrlPayResultEndpointSuccess struct {
	Data LnUrlPaySuccessData
}

func (e LnUrlPayResultEndpointSuccess) Destroy() {
	FfiDestroyerLnUrlPaySuccessData{}.Destroy(e.Data)
}

type LnUrlPayResultEndpointError struct {
	Data LnUrlErrorData
}

func (e LnUrlPayResultEndpointError) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type LnUrlPayResultPayError struct {
	Data LnUrlPayErrorData
}

func (e LnUrlPayResultPayError) Destroy() {
	FfiDestroyerLnUrlPayErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlPayResult struct{}

var FfiConverterLnUrlPayResultINSTANCE = FfiConverterLnUrlPayResult{}

func (c FfiConverterLnUrlPayResult) Lift(rb RustBufferI) LnUrlPayResult {
	return LiftFromRustBuffer[LnUrlPayResult](c, rb)
}

func (c FfiConverterLnUrlPayResult) Lower(value LnUrlPayResult) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayResult](c, value)
}
func (FfiConverterLnUrlPayResult) Read(reader io.Reader) LnUrlPayResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlPayResultEndpointSuccess{
			FfiConverterLnUrlPaySuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlPayResultEndpointError{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlPayResultPayError{
			FfiConverterLnUrlPayErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlPayResult.Read()", id))
	}
}

func (FfiConverterLnUrlPayResult) Write(writer io.Writer, value LnUrlPayResult) {
	switch variant_value := value.(type) {
	case LnUrlPayResultEndpointSuccess:
		writeInt32(writer, 1)
		FfiConverterLnUrlPaySuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultEndpointError:
		writeInt32(writer, 2)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultPayError:
		writeInt32(writer, 3)
		FfiConverterLnUrlPayErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlPayResult.Write", value))
	}
}

type FfiDestroyerLnUrlPayResult struct{}

func (_ FfiDestroyerLnUrlPayResult) Destroy(value LnUrlPayResult) {
	value.Destroy()
}

type LnUrlWithdrawError struct {
	err error
}

// Convience method to turn *LnUrlWithdrawError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlWithdrawError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlWithdrawError) Error() string {
	return fmt.Sprintf("LnUrlWithdrawError: %s", err.err.Error())
}

func (err LnUrlWithdrawError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlWithdrawErrorGeneric = fmt.Errorf("LnUrlWithdrawErrorGeneric")
var ErrLnUrlWithdrawErrorInvalidAmount = fmt.Errorf("LnUrlWithdrawErrorInvalidAmount")
var ErrLnUrlWithdrawErrorInvalidInvoice = fmt.Errorf("LnUrlWithdrawErrorInvalidInvoice")
var ErrLnUrlWithdrawErrorInvalidUri = fmt.Errorf("LnUrlWithdrawErrorInvalidUri")
var ErrLnUrlWithdrawErrorServiceConnectivity = fmt.Errorf("LnUrlWithdrawErrorServiceConnectivity")
var ErrLnUrlWithdrawErrorInvoiceNoRoutingHints = fmt.Errorf("LnUrlWithdrawErrorInvoiceNoRoutingHints")

// Variant structs
type LnUrlWithdrawErrorGeneric struct {
	Err string
}

func NewLnUrlWithdrawErrorGeneric(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorGeneric{
		Err: err}}
}

func (e LnUrlWithdrawErrorGeneric) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorGeneric) Error() string {
	return fmt.Sprint("Generic",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorGeneric
}

type LnUrlWithdrawErrorInvalidAmount struct {
	Err string
}

func NewLnUrlWithdrawErrorInvalidAmount(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidAmount{
		Err: err}}
}

func (e LnUrlWithdrawErrorInvalidAmount) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorInvalidAmount) Error() string {
	return fmt.Sprint("InvalidAmount",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorInvalidAmount) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidAmount
}

type LnUrlWithdrawErrorInvalidInvoice struct {
	Err string
}

func NewLnUrlWithdrawErrorInvalidInvoice(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidInvoice{
		Err: err}}
}

func (e LnUrlWithdrawErrorInvalidInvoice) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorInvalidInvoice) Error() string {
	return fmt.Sprint("InvalidInvoice",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorInvalidInvoice) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidInvoice
}

type LnUrlWithdrawErrorInvalidUri struct {
	Err string
}

func NewLnUrlWithdrawErrorInvalidUri(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidUri{
		Err: err}}
}

func (e LnUrlWithdrawErrorInvalidUri) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorInvalidUri) Error() string {
	return fmt.Sprint("InvalidUri",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidUri
}

type LnUrlWithdrawErrorServiceConnectivity struct {
	Err string
}

func NewLnUrlWithdrawErrorServiceConnectivity(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorServiceConnectivity{
		Err: err}}
}

func (e LnUrlWithdrawErrorServiceConnectivity) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorServiceConnectivity) Error() string {
	return fmt.Sprint("ServiceConnectivity",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorServiceConnectivity
}

type LnUrlWithdrawErrorInvoiceNoRoutingHints struct {
	Err string
}

func NewLnUrlWithdrawErrorInvoiceNoRoutingHints(
	err string,
) *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvoiceNoRoutingHints{
		Err: err}}
}

func (e LnUrlWithdrawErrorInvoiceNoRoutingHints) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err LnUrlWithdrawErrorInvoiceNoRoutingHints) Error() string {
	return fmt.Sprint("InvoiceNoRoutingHints",
		": ",

		"Err=",
		err.Err,
	)
}

func (self LnUrlWithdrawErrorInvoiceNoRoutingHints) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvoiceNoRoutingHints
}

type FfiConverterLnUrlWithdrawError struct{}

var FfiConverterLnUrlWithdrawErrorINSTANCE = FfiConverterLnUrlWithdrawError{}

func (c FfiConverterLnUrlWithdrawError) Lift(eb RustBufferI) *LnUrlWithdrawError {
	return LiftFromRustBuffer[*LnUrlWithdrawError](c, eb)
}

func (c FfiConverterLnUrlWithdrawError) Lower(value *LnUrlWithdrawError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlWithdrawError](c, value)
}

func (c FfiConverterLnUrlWithdrawError) Read(reader io.Reader) *LnUrlWithdrawError {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 2:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidAmount{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 3:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidInvoice{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 4:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidUri{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 5:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorServiceConnectivity{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 6:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvoiceNoRoutingHints{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlWithdrawError.Read()", errorID))
	}
}

func (c FfiConverterLnUrlWithdrawError) Write(writer io.Writer, value *LnUrlWithdrawError) {
	switch variantValue := value.err.(type) {
	case *LnUrlWithdrawErrorGeneric:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlWithdrawErrorInvalidAmount:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlWithdrawErrorInvalidInvoice:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlWithdrawErrorInvalidUri:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlWithdrawErrorServiceConnectivity:
		writeInt32(writer, 5)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlWithdrawErrorInvoiceNoRoutingHints:
		writeInt32(writer, 6)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlWithdrawError.Write", value))
	}
}

type FfiDestroyerLnUrlWithdrawError struct{}

func (_ FfiDestroyerLnUrlWithdrawError) Destroy(value *LnUrlWithdrawError) {
	switch variantValue := value.err.(type) {
	case LnUrlWithdrawErrorGeneric:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidAmount:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidInvoice:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidUri:
		variantValue.destroy()
	case LnUrlWithdrawErrorServiceConnectivity:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvoiceNoRoutingHints:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlWithdrawError.Destroy", value))
	}
}

type LnUrlWithdrawResult interface {
	Destroy()
}
type LnUrlWithdrawResultOk struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultOk) Destroy() {
	FfiDestroyerLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultTimeout struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultTimeout) Destroy() {
	FfiDestroyerLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlWithdrawResultErrorStatus) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlWithdrawResult struct{}

var FfiConverterLnUrlWithdrawResultINSTANCE = FfiConverterLnUrlWithdrawResult{}

func (c FfiConverterLnUrlWithdrawResult) Lift(rb RustBufferI) LnUrlWithdrawResult {
	return LiftFromRustBuffer[LnUrlWithdrawResult](c, rb)
}

func (c FfiConverterLnUrlWithdrawResult) Lower(value LnUrlWithdrawResult) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawResult](c, value)
}
func (FfiConverterLnUrlWithdrawResult) Read(reader io.Reader) LnUrlWithdrawResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlWithdrawResultOk{
			FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlWithdrawResultTimeout{
			FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlWithdrawResultErrorStatus{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlWithdrawResult.Read()", id))
	}
}

func (FfiConverterLnUrlWithdrawResult) Write(writer io.Writer, value LnUrlWithdrawResult) {
	switch variant_value := value.(type) {
	case LnUrlWithdrawResultOk:
		writeInt32(writer, 1)
		FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultTimeout:
		writeInt32(writer, 2)
		FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultErrorStatus:
		writeInt32(writer, 3)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlWithdrawResult.Write", value))
	}
}

type FfiDestroyerLnUrlWithdrawResult struct{}

func (_ FfiDestroyerLnUrlWithdrawResult) Destroy(value LnUrlWithdrawResult) {
	value.Destroy()
}

type Network uint

const (
	NetworkBitcoin Network = 1
	NetworkTestnet Network = 2
	NetworkSignet  Network = 3
	NetworkRegtest Network = 4
)

type FfiConverterNetwork struct{}

var FfiConverterNetworkINSTANCE = FfiConverterNetwork{}

func (c FfiConverterNetwork) Lift(rb RustBufferI) Network {
	return LiftFromRustBuffer[Network](c, rb)
}

func (c FfiConverterNetwork) Lower(value Network) C.RustBuffer {
	return LowerIntoRustBuffer[Network](c, value)
}
func (FfiConverterNetwork) Read(reader io.Reader) Network {
	id := readInt32(reader)
	return Network(id)
}

func (FfiConverterNetwork) Write(writer io.Writer, value Network) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerNetwork struct{}

func (_ FfiDestroyerNetwork) Destroy(value Network) {
}

type PayAmount interface {
	Destroy()
}
type PayAmountBitcoin struct {
	ReceiverAmountSat uint64
}

func (e PayAmountBitcoin) Destroy() {
	FfiDestroyerUint64{}.Destroy(e.ReceiverAmountSat)
}

type PayAmountAsset struct {
	AssetId           string
	ReceiverAmount    float64
	EstimateAssetFees *bool
}

func (e PayAmountAsset) Destroy() {
	FfiDestroyerString{}.Destroy(e.AssetId)
	FfiDestroyerFloat64{}.Destroy(e.ReceiverAmount)
	FfiDestroyerOptionalBool{}.Destroy(e.EstimateAssetFees)
}

type PayAmountDrain struct {
}

func (e PayAmountDrain) Destroy() {
}

type FfiConverterPayAmount struct{}

var FfiConverterPayAmountINSTANCE = FfiConverterPayAmount{}

func (c FfiConverterPayAmount) Lift(rb RustBufferI) PayAmount {
	return LiftFromRustBuffer[PayAmount](c, rb)
}

func (c FfiConverterPayAmount) Lower(value PayAmount) C.RustBuffer {
	return LowerIntoRustBuffer[PayAmount](c, value)
}
func (FfiConverterPayAmount) Read(reader io.Reader) PayAmount {
	id := readInt32(reader)
	switch id {
	case 1:
		return PayAmountBitcoin{
			FfiConverterUint64INSTANCE.Read(reader),
		}
	case 2:
		return PayAmountAsset{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterFloat64INSTANCE.Read(reader),
			FfiConverterOptionalBoolINSTANCE.Read(reader),
		}
	case 3:
		return PayAmountDrain{}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterPayAmount.Read()", id))
	}
}

func (FfiConverterPayAmount) Write(writer io.Writer, value PayAmount) {
	switch variant_value := value.(type) {
	case PayAmountBitcoin:
		writeInt32(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.ReceiverAmountSat)
	case PayAmountAsset:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterFloat64INSTANCE.Write(writer, variant_value.ReceiverAmount)
		FfiConverterOptionalBoolINSTANCE.Write(writer, variant_value.EstimateAssetFees)
	case PayAmountDrain:
		writeInt32(writer, 3)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterPayAmount.Write", value))
	}
}

type FfiDestroyerPayAmount struct{}

func (_ FfiDestroyerPayAmount) Destroy(value PayAmount) {
	value.Destroy()
}

type PaymentDetails interface {
	Destroy()
}
type PaymentDetailsLightning struct {
	SwapId                      string
	Description                 string
	LiquidExpirationBlockheight uint32
	Preimage                    *string
	Invoice                     *string
	Bolt12Offer                 *string
	PaymentHash                 *string
	DestinationPubkey           *string
	LnurlInfo                   *LnUrlInfo
	Bip353Address               *string
	PayerNote                   *string
	ClaimTxId                   *string
	RefundTxId                  *string
	RefundTxAmountSat           *uint64
}

func (e PaymentDetailsLightning) Destroy() {
	FfiDestroyerString{}.Destroy(e.SwapId)
	FfiDestroyerString{}.Destroy(e.Description)
	FfiDestroyerUint32{}.Destroy(e.LiquidExpirationBlockheight)
	FfiDestroyerOptionalString{}.Destroy(e.Preimage)
	FfiDestroyerOptionalString{}.Destroy(e.Invoice)
	FfiDestroyerOptionalString{}.Destroy(e.Bolt12Offer)
	FfiDestroyerOptionalString{}.Destroy(e.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(e.DestinationPubkey)
	FfiDestroyerOptionalLnUrlInfo{}.Destroy(e.LnurlInfo)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
	FfiDestroyerOptionalString{}.Destroy(e.PayerNote)
	FfiDestroyerOptionalString{}.Destroy(e.ClaimTxId)
	FfiDestroyerOptionalString{}.Destroy(e.RefundTxId)
	FfiDestroyerOptionalUint64{}.Destroy(e.RefundTxAmountSat)
}

type PaymentDetailsLiquid struct {
	AssetId       string
	Destination   string
	Description   string
	AssetInfo     *AssetInfo
	LnurlInfo     *LnUrlInfo
	Bip353Address *string
	PayerNote     *string
}

func (e PaymentDetailsLiquid) Destroy() {
	FfiDestroyerString{}.Destroy(e.AssetId)
	FfiDestroyerString{}.Destroy(e.Destination)
	FfiDestroyerString{}.Destroy(e.Description)
	FfiDestroyerOptionalAssetInfo{}.Destroy(e.AssetInfo)
	FfiDestroyerOptionalLnUrlInfo{}.Destroy(e.LnurlInfo)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
	FfiDestroyerOptionalString{}.Destroy(e.PayerNote)
}

type PaymentDetailsBitcoin struct {
	SwapId                       string
	BitcoinAddress               string
	Description                  string
	AutoAcceptedFees             bool
	BitcoinExpirationBlockheight *uint32
	LiquidExpirationBlockheight  *uint32
	LockupTxId                   *string
	ClaimTxId                    *string
	RefundTxId                   *string
	RefundTxAmountSat            *uint64
}

func (e PaymentDetailsBitcoin) Destroy() {
	FfiDestroyerString{}.Destroy(e.SwapId)
	FfiDestroyerString{}.Destroy(e.BitcoinAddress)
	FfiDestroyerString{}.Destroy(e.Description)
	FfiDestroyerBool{}.Destroy(e.AutoAcceptedFees)
	FfiDestroyerOptionalUint32{}.Destroy(e.BitcoinExpirationBlockheight)
	FfiDestroyerOptionalUint32{}.Destroy(e.LiquidExpirationBlockheight)
	FfiDestroyerOptionalString{}.Destroy(e.LockupTxId)
	FfiDestroyerOptionalString{}.Destroy(e.ClaimTxId)
	FfiDestroyerOptionalString{}.Destroy(e.RefundTxId)
	FfiDestroyerOptionalUint64{}.Destroy(e.RefundTxAmountSat)
}

type FfiConverterPaymentDetails struct{}

var FfiConverterPaymentDetailsINSTANCE = FfiConverterPaymentDetails{}

func (c FfiConverterPaymentDetails) Lift(rb RustBufferI) PaymentDetails {
	return LiftFromRustBuffer[PaymentDetails](c, rb)
}

func (c FfiConverterPaymentDetails) Lower(value PaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentDetails](c, value)
}
func (FfiConverterPaymentDetails) Read(reader io.Reader) PaymentDetails {
	id := readInt32(reader)
	switch id {
	case 1:
		return PaymentDetailsLightning{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterUint32INSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalLnUrlInfoINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalUint64INSTANCE.Read(reader),
		}
	case 2:
		return PaymentDetailsLiquid{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterOptionalAssetInfoINSTANCE.Read(reader),
			FfiConverterOptionalLnUrlInfoINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 3:
		return PaymentDetailsBitcoin{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterBoolINSTANCE.Read(reader),
			FfiConverterOptionalUint32INSTANCE.Read(reader),
			FfiConverterOptionalUint32INSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalUint64INSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterPaymentDetails.Read()", id))
	}
}

func (FfiConverterPaymentDetails) Write(writer io.Writer, value PaymentDetails) {
	switch variant_value := value.(type) {
	case PaymentDetailsLightning:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variant_value.SwapId)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Description)
		FfiConverterUint32INSTANCE.Write(writer, variant_value.LiquidExpirationBlockheight)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Preimage)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Invoice)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bolt12Offer)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.PaymentHash)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.DestinationPubkey)
		FfiConverterOptionalLnUrlInfoINSTANCE.Write(writer, variant_value.LnurlInfo)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.PayerNote)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.ClaimTxId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.RefundTxId)
		FfiConverterOptionalUint64INSTANCE.Write(writer, variant_value.RefundTxAmountSat)
	case PaymentDetailsLiquid:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Destination)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Description)
		FfiConverterOptionalAssetInfoINSTANCE.Write(writer, variant_value.AssetInfo)
		FfiConverterOptionalLnUrlInfoINSTANCE.Write(writer, variant_value.LnurlInfo)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.PayerNote)
	case PaymentDetailsBitcoin:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variant_value.SwapId)
		FfiConverterStringINSTANCE.Write(writer, variant_value.BitcoinAddress)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Description)
		FfiConverterBoolINSTANCE.Write(writer, variant_value.AutoAcceptedFees)
		FfiConverterOptionalUint32INSTANCE.Write(writer, variant_value.BitcoinExpirationBlockheight)
		FfiConverterOptionalUint32INSTANCE.Write(writer, variant_value.LiquidExpirationBlockheight)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.LockupTxId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.ClaimTxId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.RefundTxId)
		FfiConverterOptionalUint64INSTANCE.Write(writer, variant_value.RefundTxAmountSat)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterPaymentDetails.Write", value))
	}
}

type FfiDestroyerPaymentDetails struct{}

func (_ FfiDestroyerPaymentDetails) Destroy(value PaymentDetails) {
	value.Destroy()
}

type PaymentError struct {
	err error
}

// Convience method to turn *PaymentError into error
// Avoiding treating nil pointer as non nil error interface
func (err *PaymentError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err PaymentError) Error() string {
	return fmt.Sprintf("PaymentError: %s", err.err.Error())
}

func (err PaymentError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrPaymentErrorAlreadyClaimed = fmt.Errorf("PaymentErrorAlreadyClaimed")
var ErrPaymentErrorAlreadyPaid = fmt.Errorf("PaymentErrorAlreadyPaid")
var ErrPaymentErrorPaymentInProgress = fmt.Errorf("PaymentErrorPaymentInProgress")
var ErrPaymentErrorAmountOutOfRange = fmt.Errorf("PaymentErrorAmountOutOfRange")
var ErrPaymentErrorAmountMissing = fmt.Errorf("PaymentErrorAmountMissing")
var ErrPaymentErrorAssetError = fmt.Errorf("PaymentErrorAssetError")
var ErrPaymentErrorGeneric = fmt.Errorf("PaymentErrorGeneric")
var ErrPaymentErrorInvalidOrExpiredFees = fmt.Errorf("PaymentErrorInvalidOrExpiredFees")
var ErrPaymentErrorInsufficientFunds = fmt.Errorf("PaymentErrorInsufficientFunds")
var ErrPaymentErrorInvalidDescription = fmt.Errorf("PaymentErrorInvalidDescription")
var ErrPaymentErrorInvalidInvoice = fmt.Errorf("PaymentErrorInvalidInvoice")
var ErrPaymentErrorInvalidNetwork = fmt.Errorf("PaymentErrorInvalidNetwork")
var ErrPaymentErrorInvalidPreimage = fmt.Errorf("PaymentErrorInvalidPreimage")
var ErrPaymentErrorPairsNotFound = fmt.Errorf("PaymentErrorPairsNotFound")
var ErrPaymentErrorPaymentTimeout = fmt.Errorf("PaymentErrorPaymentTimeout")
var ErrPaymentErrorPersistError = fmt.Errorf("PaymentErrorPersistError")
var ErrPaymentErrorReceiveError = fmt.Errorf("PaymentErrorReceiveError")
var ErrPaymentErrorRefunded = fmt.Errorf("PaymentErrorRefunded")
var ErrPaymentErrorSelfTransferNotSupported = fmt.Errorf("PaymentErrorSelfTransferNotSupported")
var ErrPaymentErrorSendError = fmt.Errorf("PaymentErrorSendError")
var ErrPaymentErrorSignerError = fmt.Errorf("PaymentErrorSignerError")

// Variant structs
type PaymentErrorAlreadyClaimed struct {
	message string
}

func NewPaymentErrorAlreadyClaimed() *PaymentError {
	return &PaymentError{err: &PaymentErrorAlreadyClaimed{}}
}

func (e PaymentErrorAlreadyClaimed) destroy() {
}

func (err PaymentErrorAlreadyClaimed) Error() string {
	return fmt.Sprintf("AlreadyClaimed: %s", err.message)
}

func (self PaymentErrorAlreadyClaimed) Is(target error) bool {
	return target == ErrPaymentErrorAlreadyClaimed
}

type PaymentErrorAlreadyPaid struct {
	message string
}

func NewPaymentErrorAlreadyPaid() *PaymentError {
	return &PaymentError{err: &PaymentErrorAlreadyPaid{}}
}

func (e PaymentErrorAlreadyPaid) destroy() {
}

func (err PaymentErrorAlreadyPaid) Error() string {
	return fmt.Sprintf("AlreadyPaid: %s", err.message)
}

func (self PaymentErrorAlreadyPaid) Is(target error) bool {
	return target == ErrPaymentErrorAlreadyPaid
}

type PaymentErrorPaymentInProgress struct {
	message string
}

func NewPaymentErrorPaymentInProgress() *PaymentError {
	return &PaymentError{err: &PaymentErrorPaymentInProgress{}}
}

func (e PaymentErrorPaymentInProgress) destroy() {
}

func (err PaymentErrorPaymentInProgress) Error() string {
	return fmt.Sprintf("PaymentInProgress: %s", err.message)
}

func (self PaymentErrorPaymentInProgress) Is(target error) bool {
	return target == ErrPaymentErrorPaymentInProgress
}

type PaymentErrorAmountOutOfRange struct {
	message string
}

func NewPaymentErrorAmountOutOfRange() *PaymentError {
	return &PaymentError{err: &PaymentErrorAmountOutOfRange{}}
}

func (e PaymentErrorAmountOutOfRange) destroy() {
}

func (err PaymentErrorAmountOutOfRange) Error() string {
	return fmt.Sprintf("AmountOutOfRange: %s", err.message)
}

func (self PaymentErrorAmountOutOfRange) Is(target error) bool {
	return target == ErrPaymentErrorAmountOutOfRange
}

type PaymentErrorAmountMissing struct {
	message string
}

func NewPaymentErrorAmountMissing() *PaymentError {
	return &PaymentError{err: &PaymentErrorAmountMissing{}}
}

func (e PaymentErrorAmountMissing) destroy() {
}

func (err PaymentErrorAmountMissing) Error() string {
	return fmt.Sprintf("AmountMissing: %s", err.message)
}

func (self PaymentErrorAmountMissing) Is(target error) bool {
	return target == ErrPaymentErrorAmountMissing
}

type PaymentErrorAssetError struct {
	message string
}

func NewPaymentErrorAssetError() *PaymentError {
	return &PaymentError{err: &PaymentErrorAssetError{}}
}

func (e PaymentErrorAssetError) destroy() {
}

func (err PaymentErrorAssetError) Error() string {
	return fmt.Sprintf("AssetError: %s", err.message)
}

func (self PaymentErrorAssetError) Is(target error) bool {
	return target == ErrPaymentErrorAssetError
}

type PaymentErrorGeneric struct {
	message string
}

func NewPaymentErrorGeneric() *PaymentError {
	return &PaymentError{err: &PaymentErrorGeneric{}}
}

func (e PaymentErrorGeneric) destroy() {
}

func (err PaymentErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self PaymentErrorGeneric) Is(target error) bool {
	return target == ErrPaymentErrorGeneric
}

type PaymentErrorInvalidOrExpiredFees struct {
	message string
}

func NewPaymentErrorInvalidOrExpiredFees() *PaymentError {
	return &PaymentError{err: &PaymentErrorInvalidOrExpiredFees{}}
}

func (e PaymentErrorInvalidOrExpiredFees) destroy() {
}

func (err PaymentErrorInvalidOrExpiredFees) Error() string {
	return fmt.Sprintf("InvalidOrExpiredFees: %s", err.message)
}

func (self PaymentErrorInvalidOrExpiredFees) Is(target error) bool {
	return target == ErrPaymentErrorInvalidOrExpiredFees
}

type PaymentErrorInsufficientFunds struct {
	message string
}

func NewPaymentErrorInsufficientFunds() *PaymentError {
	return &PaymentError{err: &PaymentErrorInsufficientFunds{}}
}

func (e PaymentErrorInsufficientFunds) destroy() {
}

func (err PaymentErrorInsufficientFunds) Error() string {
	return fmt.Sprintf("InsufficientFunds: %s", err.message)
}

func (self PaymentErrorInsufficientFunds) Is(target error) bool {
	return target == ErrPaymentErrorInsufficientFunds
}

type PaymentErrorInvalidDescription struct {
	message string
}

func NewPaymentErrorInvalidDescription() *PaymentError {
	return &PaymentError{err: &PaymentErrorInvalidDescription{}}
}

func (e PaymentErrorInvalidDescription) destroy() {
}

func (err PaymentErrorInvalidDescription) Error() string {
	return fmt.Sprintf("InvalidDescription: %s", err.message)
}

func (self PaymentErrorInvalidDescription) Is(target error) bool {
	return target == ErrPaymentErrorInvalidDescription
}

type PaymentErrorInvalidInvoice struct {
	message string
}

func NewPaymentErrorInvalidInvoice() *PaymentError {
	return &PaymentError{err: &PaymentErrorInvalidInvoice{}}
}

func (e PaymentErrorInvalidInvoice) destroy() {
}

func (err PaymentErrorInvalidInvoice) Error() string {
	return fmt.Sprintf("InvalidInvoice: %s", err.message)
}

func (self PaymentErrorInvalidInvoice) Is(target error) bool {
	return target == ErrPaymentErrorInvalidInvoice
}

type PaymentErrorInvalidNetwork struct {
	message string
}

func NewPaymentErrorInvalidNetwork() *PaymentError {
	return &PaymentError{err: &PaymentErrorInvalidNetwork{}}
}

func (e PaymentErrorInvalidNetwork) destroy() {
}

func (err PaymentErrorInvalidNetwork) Error() string {
	return fmt.Sprintf("InvalidNetwork: %s", err.message)
}

func (self PaymentErrorInvalidNetwork) Is(target error) bool {
	return target == ErrPaymentErrorInvalidNetwork
}

type PaymentErrorInvalidPreimage struct {
	message string
}

func NewPaymentErrorInvalidPreimage() *PaymentError {
	return &PaymentError{err: &PaymentErrorInvalidPreimage{}}
}

func (e PaymentErrorInvalidPreimage) destroy() {
}

func (err PaymentErrorInvalidPreimage) Error() string {
	return fmt.Sprintf("InvalidPreimage: %s", err.message)
}

func (self PaymentErrorInvalidPreimage) Is(target error) bool {
	return target == ErrPaymentErrorInvalidPreimage
}

type PaymentErrorPairsNotFound struct {
	message string
}

func NewPaymentErrorPairsNotFound() *PaymentError {
	return &PaymentError{err: &PaymentErrorPairsNotFound{}}
}

func (e PaymentErrorPairsNotFound) destroy() {
}

func (err PaymentErrorPairsNotFound) Error() string {
	return fmt.Sprintf("PairsNotFound: %s", err.message)
}

func (self PaymentErrorPairsNotFound) Is(target error) bool {
	return target == ErrPaymentErrorPairsNotFound
}

type PaymentErrorPaymentTimeout struct {
	message string
}

func NewPaymentErrorPaymentTimeout() *PaymentError {
	return &PaymentError{err: &PaymentErrorPaymentTimeout{}}
}

func (e PaymentErrorPaymentTimeout) destroy() {
}

func (err PaymentErrorPaymentTimeout) Error() string {
	return fmt.Sprintf("PaymentTimeout: %s", err.message)
}

func (self PaymentErrorPaymentTimeout) Is(target error) bool {
	return target == ErrPaymentErrorPaymentTimeout
}

type PaymentErrorPersistError struct {
	message string
}

func NewPaymentErrorPersistError() *PaymentError {
	return &PaymentError{err: &PaymentErrorPersistError{}}
}

func (e PaymentErrorPersistError) destroy() {
}

func (err PaymentErrorPersistError) Error() string {
	return fmt.Sprintf("PersistError: %s", err.message)
}

func (self PaymentErrorPersistError) Is(target error) bool {
	return target == ErrPaymentErrorPersistError
}

type PaymentErrorReceiveError struct {
	message string
}

func NewPaymentErrorReceiveError() *PaymentError {
	return &PaymentError{err: &PaymentErrorReceiveError{}}
}

func (e PaymentErrorReceiveError) destroy() {
}

func (err PaymentErrorReceiveError) Error() string {
	return fmt.Sprintf("ReceiveError: %s", err.message)
}

func (self PaymentErrorReceiveError) Is(target error) bool {
	return target == ErrPaymentErrorReceiveError
}

type PaymentErrorRefunded struct {
	message string
}

func NewPaymentErrorRefunded() *PaymentError {
	return &PaymentError{err: &PaymentErrorRefunded{}}
}

func (e PaymentErrorRefunded) destroy() {
}

func (err PaymentErrorRefunded) Error() string {
	return fmt.Sprintf("Refunded: %s", err.message)
}

func (self PaymentErrorRefunded) Is(target error) bool {
	return target == ErrPaymentErrorRefunded
}

type PaymentErrorSelfTransferNotSupported struct {
	message string
}

func NewPaymentErrorSelfTransferNotSupported() *PaymentError {
	return &PaymentError{err: &PaymentErrorSelfTransferNotSupported{}}
}

func (e PaymentErrorSelfTransferNotSupported) destroy() {
}

func (err PaymentErrorSelfTransferNotSupported) Error() string {
	return fmt.Sprintf("SelfTransferNotSupported: %s", err.message)
}

func (self PaymentErrorSelfTransferNotSupported) Is(target error) bool {
	return target == ErrPaymentErrorSelfTransferNotSupported
}

type PaymentErrorSendError struct {
	message string
}

func NewPaymentErrorSendError() *PaymentError {
	return &PaymentError{err: &PaymentErrorSendError{}}
}

func (e PaymentErrorSendError) destroy() {
}

func (err PaymentErrorSendError) Error() string {
	return fmt.Sprintf("SendError: %s", err.message)
}

func (self PaymentErrorSendError) Is(target error) bool {
	return target == ErrPaymentErrorSendError
}

type PaymentErrorSignerError struct {
	message string
}

func NewPaymentErrorSignerError() *PaymentError {
	return &PaymentError{err: &PaymentErrorSignerError{}}
}

func (e PaymentErrorSignerError) destroy() {
}

func (err PaymentErrorSignerError) Error() string {
	return fmt.Sprintf("SignerError: %s", err.message)
}

func (self PaymentErrorSignerError) Is(target error) bool {
	return target == ErrPaymentErrorSignerError
}

type FfiConverterPaymentError struct{}

var FfiConverterPaymentErrorINSTANCE = FfiConverterPaymentError{}

func (c FfiConverterPaymentError) Lift(eb RustBufferI) *PaymentError {
	return LiftFromRustBuffer[*PaymentError](c, eb)
}

func (c FfiConverterPaymentError) Lower(value *PaymentError) C.RustBuffer {
	return LowerIntoRustBuffer[*PaymentError](c, value)
}

func (c FfiConverterPaymentError) Read(reader io.Reader) *PaymentError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &PaymentError{&PaymentErrorAlreadyClaimed{message}}
	case 2:
		return &PaymentError{&PaymentErrorAlreadyPaid{message}}
	case 3:
		return &PaymentError{&PaymentErrorPaymentInProgress{message}}
	case 4:
		return &PaymentError{&PaymentErrorAmountOutOfRange{message}}
	case 5:
		return &PaymentError{&PaymentErrorAmountMissing{message}}
	case 6:
		return &PaymentError{&PaymentErrorAssetError{message}}
	case 7:
		return &PaymentError{&PaymentErrorGeneric{message}}
	case 8:
		return &PaymentError{&PaymentErrorInvalidOrExpiredFees{message}}
	case 9:
		return &PaymentError{&PaymentErrorInsufficientFunds{message}}
	case 10:
		return &PaymentError{&PaymentErrorInvalidDescription{message}}
	case 11:
		return &PaymentError{&PaymentErrorInvalidInvoice{message}}
	case 12:
		return &PaymentError{&PaymentErrorInvalidNetwork{message}}
	case 13:
		return &PaymentError{&PaymentErrorInvalidPreimage{message}}
	case 14:
		return &PaymentError{&PaymentErrorPairsNotFound{message}}
	case 15:
		return &PaymentError{&PaymentErrorPaymentTimeout{message}}
	case 16:
		return &PaymentError{&PaymentErrorPersistError{message}}
	case 17:
		return &PaymentError{&PaymentErrorReceiveError{message}}
	case 18:
		return &PaymentError{&PaymentErrorRefunded{message}}
	case 19:
		return &PaymentError{&PaymentErrorSelfTransferNotSupported{message}}
	case 20:
		return &PaymentError{&PaymentErrorSendError{message}}
	case 21:
		return &PaymentError{&PaymentErrorSignerError{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterPaymentError.Read()", errorID))
	}

}

func (c FfiConverterPaymentError) Write(writer io.Writer, value *PaymentError) {
	switch variantValue := value.err.(type) {
	case *PaymentErrorAlreadyClaimed:
		writeInt32(writer, 1)
	case *PaymentErrorAlreadyPaid:
		writeInt32(writer, 2)
	case *PaymentErrorPaymentInProgress:
		writeInt32(writer, 3)
	case *PaymentErrorAmountOutOfRange:
		writeInt32(writer, 4)
	case *PaymentErrorAmountMissing:
		writeInt32(writer, 5)
	case *PaymentErrorAssetError:
		writeInt32(writer, 6)
	case *PaymentErrorGeneric:
		writeInt32(writer, 7)
	case *PaymentErrorInvalidOrExpiredFees:
		writeInt32(writer, 8)
	case *PaymentErrorInsufficientFunds:
		writeInt32(writer, 9)
	case *PaymentErrorInvalidDescription:
		writeInt32(writer, 10)
	case *PaymentErrorInvalidInvoice:
		writeInt32(writer, 11)
	case *PaymentErrorInvalidNetwork:
		writeInt32(writer, 12)
	case *PaymentErrorInvalidPreimage:
		writeInt32(writer, 13)
	case *PaymentErrorPairsNotFound:
		writeInt32(writer, 14)
	case *PaymentErrorPaymentTimeout:
		writeInt32(writer, 15)
	case *PaymentErrorPersistError:
		writeInt32(writer, 16)
	case *PaymentErrorReceiveError:
		writeInt32(writer, 17)
	case *PaymentErrorRefunded:
		writeInt32(writer, 18)
	case *PaymentErrorSelfTransferNotSupported:
		writeInt32(writer, 19)
	case *PaymentErrorSendError:
		writeInt32(writer, 20)
	case *PaymentErrorSignerError:
		writeInt32(writer, 21)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterPaymentError.Write", value))
	}
}

type FfiDestroyerPaymentError struct{}

func (_ FfiDestroyerPaymentError) Destroy(value *PaymentError) {
	switch variantValue := value.err.(type) {
	case PaymentErrorAlreadyClaimed:
		variantValue.destroy()
	case PaymentErrorAlreadyPaid:
		variantValue.destroy()
	case PaymentErrorPaymentInProgress:
		variantValue.destroy()
	case PaymentErrorAmountOutOfRange:
		variantValue.destroy()
	case PaymentErrorAmountMissing:
		variantValue.destroy()
	case PaymentErrorAssetError:
		variantValue.destroy()
	case PaymentErrorGeneric:
		variantValue.destroy()
	case PaymentErrorInvalidOrExpiredFees:
		variantValue.destroy()
	case PaymentErrorInsufficientFunds:
		variantValue.destroy()
	case PaymentErrorInvalidDescription:
		variantValue.destroy()
	case PaymentErrorInvalidInvoice:
		variantValue.destroy()
	case PaymentErrorInvalidNetwork:
		variantValue.destroy()
	case PaymentErrorInvalidPreimage:
		variantValue.destroy()
	case PaymentErrorPairsNotFound:
		variantValue.destroy()
	case PaymentErrorPaymentTimeout:
		variantValue.destroy()
	case PaymentErrorPersistError:
		variantValue.destroy()
	case PaymentErrorReceiveError:
		variantValue.destroy()
	case PaymentErrorRefunded:
		variantValue.destroy()
	case PaymentErrorSelfTransferNotSupported:
		variantValue.destroy()
	case PaymentErrorSendError:
		variantValue.destroy()
	case PaymentErrorSignerError:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerPaymentError.Destroy", value))
	}
}

type PaymentMethod uint

const (
	PaymentMethodLightning      PaymentMethod = 1
	PaymentMethodBolt11Invoice  PaymentMethod = 2
	PaymentMethodBolt12Offer    PaymentMethod = 3
	PaymentMethodBitcoinAddress PaymentMethod = 4
	PaymentMethodLiquidAddress  PaymentMethod = 5
)

type FfiConverterPaymentMethod struct{}

var FfiConverterPaymentMethodINSTANCE = FfiConverterPaymentMethod{}

func (c FfiConverterPaymentMethod) Lift(rb RustBufferI) PaymentMethod {
	return LiftFromRustBuffer[PaymentMethod](c, rb)
}

func (c FfiConverterPaymentMethod) Lower(value PaymentMethod) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentMethod](c, value)
}
func (FfiConverterPaymentMethod) Read(reader io.Reader) PaymentMethod {
	id := readInt32(reader)
	return PaymentMethod(id)
}

func (FfiConverterPaymentMethod) Write(writer io.Writer, value PaymentMethod) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentMethod struct{}

func (_ FfiDestroyerPaymentMethod) Destroy(value PaymentMethod) {
}

type PaymentState uint

const (
	PaymentStateCreated              PaymentState = 1
	PaymentStatePending              PaymentState = 2
	PaymentStateComplete             PaymentState = 3
	PaymentStateFailed               PaymentState = 4
	PaymentStateTimedOut             PaymentState = 5
	PaymentStateRefundable           PaymentState = 6
	PaymentStateRefundPending        PaymentState = 7
	PaymentStateWaitingFeeAcceptance PaymentState = 8
)

type FfiConverterPaymentState struct{}

var FfiConverterPaymentStateINSTANCE = FfiConverterPaymentState{}

func (c FfiConverterPaymentState) Lift(rb RustBufferI) PaymentState {
	return LiftFromRustBuffer[PaymentState](c, rb)
}

func (c FfiConverterPaymentState) Lower(value PaymentState) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentState](c, value)
}
func (FfiConverterPaymentState) Read(reader io.Reader) PaymentState {
	id := readInt32(reader)
	return PaymentState(id)
}

func (FfiConverterPaymentState) Write(writer io.Writer, value PaymentState) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentState struct{}

func (_ FfiDestroyerPaymentState) Destroy(value PaymentState) {
}

type PaymentType uint

const (
	PaymentTypeReceive PaymentType = 1
	PaymentTypeSend    PaymentType = 2
)

type FfiConverterPaymentType struct{}

var FfiConverterPaymentTypeINSTANCE = FfiConverterPaymentType{}

func (c FfiConverterPaymentType) Lift(rb RustBufferI) PaymentType {
	return LiftFromRustBuffer[PaymentType](c, rb)
}

func (c FfiConverterPaymentType) Lower(value PaymentType) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentType](c, value)
}
func (FfiConverterPaymentType) Read(reader io.Reader) PaymentType {
	id := readInt32(reader)
	return PaymentType(id)
}

func (FfiConverterPaymentType) Write(writer io.Writer, value PaymentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentType struct{}

func (_ FfiDestroyerPaymentType) Destroy(value PaymentType) {
}

type ReceiveAmount interface {
	Destroy()
}
type ReceiveAmountBitcoin struct {
	PayerAmountSat uint64
}

func (e ReceiveAmountBitcoin) Destroy() {
	FfiDestroyerUint64{}.Destroy(e.PayerAmountSat)
}

type ReceiveAmountAsset struct {
	AssetId     string
	PayerAmount *float64
}

func (e ReceiveAmountAsset) Destroy() {
	FfiDestroyerString{}.Destroy(e.AssetId)
	FfiDestroyerOptionalFloat64{}.Destroy(e.PayerAmount)
}

type FfiConverterReceiveAmount struct{}

var FfiConverterReceiveAmountINSTANCE = FfiConverterReceiveAmount{}

func (c FfiConverterReceiveAmount) Lift(rb RustBufferI) ReceiveAmount {
	return LiftFromRustBuffer[ReceiveAmount](c, rb)
}

func (c FfiConverterReceiveAmount) Lower(value ReceiveAmount) C.RustBuffer {
	return LowerIntoRustBuffer[ReceiveAmount](c, value)
}
func (FfiConverterReceiveAmount) Read(reader io.Reader) ReceiveAmount {
	id := readInt32(reader)
	switch id {
	case 1:
		return ReceiveAmountBitcoin{
			FfiConverterUint64INSTANCE.Read(reader),
		}
	case 2:
		return ReceiveAmountAsset{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterOptionalFloat64INSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterReceiveAmount.Read()", id))
	}
}

func (FfiConverterReceiveAmount) Write(writer io.Writer, value ReceiveAmount) {
	switch variant_value := value.(type) {
	case ReceiveAmountBitcoin:
		writeInt32(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.PayerAmountSat)
	case ReceiveAmountAsset:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterOptionalFloat64INSTANCE.Write(writer, variant_value.PayerAmount)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterReceiveAmount.Write", value))
	}
}

type FfiDestroyerReceiveAmount struct{}

func (_ FfiDestroyerReceiveAmount) Destroy(value ReceiveAmount) {
	value.Destroy()
}

// /////////////////////////////
type SdkError struct {
	err error
}

// Convience method to turn *SdkError into error
// Avoiding treating nil pointer as non nil error interface
func (err *SdkError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err SdkError) Error() string {
	return fmt.Sprintf("SdkError: %s", err.err.Error())
}

func (err SdkError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSdkErrorAlreadyStarted = fmt.Errorf("SdkErrorAlreadyStarted")
var ErrSdkErrorGeneric = fmt.Errorf("SdkErrorGeneric")
var ErrSdkErrorNotStarted = fmt.Errorf("SdkErrorNotStarted")
var ErrSdkErrorServiceConnectivity = fmt.Errorf("SdkErrorServiceConnectivity")

// Variant structs
type SdkErrorAlreadyStarted struct {
	message string
}

func NewSdkErrorAlreadyStarted() *SdkError {
	return &SdkError{err: &SdkErrorAlreadyStarted{}}
}

func (e SdkErrorAlreadyStarted) destroy() {
}

func (err SdkErrorAlreadyStarted) Error() string {
	return fmt.Sprintf("AlreadyStarted: %s", err.message)
}

func (self SdkErrorAlreadyStarted) Is(target error) bool {
	return target == ErrSdkErrorAlreadyStarted
}

type SdkErrorGeneric struct {
	message string
}

func NewSdkErrorGeneric() *SdkError {
	return &SdkError{err: &SdkErrorGeneric{}}
}

func (e SdkErrorGeneric) destroy() {
}

func (err SdkErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self SdkErrorGeneric) Is(target error) bool {
	return target == ErrSdkErrorGeneric
}

type SdkErrorNotStarted struct {
	message string
}

func NewSdkErrorNotStarted() *SdkError {
	return &SdkError{err: &SdkErrorNotStarted{}}
}

func (e SdkErrorNotStarted) destroy() {
}

func (err SdkErrorNotStarted) Error() string {
	return fmt.Sprintf("NotStarted: %s", err.message)
}

func (self SdkErrorNotStarted) Is(target error) bool {
	return target == ErrSdkErrorNotStarted
}

type SdkErrorServiceConnectivity struct {
	message string
}

func NewSdkErrorServiceConnectivity() *SdkError {
	return &SdkError{err: &SdkErrorServiceConnectivity{}}
}

func (e SdkErrorServiceConnectivity) destroy() {
}

func (err SdkErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SdkErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSdkErrorServiceConnectivity
}

type FfiConverterSdkError struct{}

var FfiConverterSdkErrorINSTANCE = FfiConverterSdkError{}

func (c FfiConverterSdkError) Lift(eb RustBufferI) *SdkError {
	return LiftFromRustBuffer[*SdkError](c, eb)
}

func (c FfiConverterSdkError) Lower(value *SdkError) C.RustBuffer {
	return LowerIntoRustBuffer[*SdkError](c, value)
}

func (c FfiConverterSdkError) Read(reader io.Reader) *SdkError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SdkError{&SdkErrorAlreadyStarted{message}}
	case 2:
		return &SdkError{&SdkErrorGeneric{message}}
	case 3:
		return &SdkError{&SdkErrorNotStarted{message}}
	case 4:
		return &SdkError{&SdkErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterSdkError.Read()", errorID))
	}

}

func (c FfiConverterSdkError) Write(writer io.Writer, value *SdkError) {
	switch variantValue := value.err.(type) {
	case *SdkErrorAlreadyStarted:
		writeInt32(writer, 1)
	case *SdkErrorGeneric:
		writeInt32(writer, 2)
	case *SdkErrorNotStarted:
		writeInt32(writer, 3)
	case *SdkErrorServiceConnectivity:
		writeInt32(writer, 4)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterSdkError.Write", value))
	}
}

type FfiDestroyerSdkError struct{}

func (_ FfiDestroyerSdkError) Destroy(value *SdkError) {
	switch variantValue := value.err.(type) {
	case SdkErrorAlreadyStarted:
		variantValue.destroy()
	case SdkErrorGeneric:
		variantValue.destroy()
	case SdkErrorNotStarted:
		variantValue.destroy()
	case SdkErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerSdkError.Destroy", value))
	}
}

type SdkEvent interface {
	Destroy()
}
type SdkEventPaymentFailed struct {
	Details Payment
}

func (e SdkEventPaymentFailed) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentPending struct {
	Details Payment
}

func (e SdkEventPaymentPending) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefundable struct {
	Details Payment
}

func (e SdkEventPaymentRefundable) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefunded struct {
	Details Payment
}

func (e SdkEventPaymentRefunded) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefundPending struct {
	Details Payment
}

func (e SdkEventPaymentRefundPending) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentSucceeded struct {
	Details Payment
}

func (e SdkEventPaymentSucceeded) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentWaitingConfirmation struct {
	Details Payment
}

func (e SdkEventPaymentWaitingConfirmation) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventPaymentWaitingFeeAcceptance struct {
	Details Payment
}

func (e SdkEventPaymentWaitingFeeAcceptance) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type SdkEventSynced struct {
}

func (e SdkEventSynced) Destroy() {
}

type SdkEventDataSynced struct {
	DidPullNewRecords bool
}

func (e SdkEventDataSynced) Destroy() {
	FfiDestroyerBool{}.Destroy(e.DidPullNewRecords)
}

type FfiConverterSdkEvent struct{}

var FfiConverterSdkEventINSTANCE = FfiConverterSdkEvent{}

func (c FfiConverterSdkEvent) Lift(rb RustBufferI) SdkEvent {
	return LiftFromRustBuffer[SdkEvent](c, rb)
}

func (c FfiConverterSdkEvent) Lower(value SdkEvent) C.RustBuffer {
	return LowerIntoRustBuffer[SdkEvent](c, value)
}
func (FfiConverterSdkEvent) Read(reader io.Reader) SdkEvent {
	id := readInt32(reader)
	switch id {
	case 1:
		return SdkEventPaymentFailed{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 2:
		return SdkEventPaymentPending{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 3:
		return SdkEventPaymentRefundable{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 4:
		return SdkEventPaymentRefunded{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 5:
		return SdkEventPaymentRefundPending{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 6:
		return SdkEventPaymentSucceeded{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 7:
		return SdkEventPaymentWaitingConfirmation{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 8:
		return SdkEventPaymentWaitingFeeAcceptance{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 9:
		return SdkEventSynced{}
	case 10:
		return SdkEventDataSynced{
			FfiConverterBoolINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterSdkEvent.Read()", id))
	}
}

func (FfiConverterSdkEvent) Write(writer io.Writer, value SdkEvent) {
	switch variant_value := value.(type) {
	case SdkEventPaymentFailed:
		writeInt32(writer, 1)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentPending:
		writeInt32(writer, 2)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefundable:
		writeInt32(writer, 3)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefunded:
		writeInt32(writer, 4)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefundPending:
		writeInt32(writer, 5)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentSucceeded:
		writeInt32(writer, 6)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentWaitingConfirmation:
		writeInt32(writer, 7)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentWaitingFeeAcceptance:
		writeInt32(writer, 8)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventSynced:
		writeInt32(writer, 9)
	case SdkEventDataSynced:
		writeInt32(writer, 10)
		FfiConverterBoolINSTANCE.Write(writer, variant_value.DidPullNewRecords)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterSdkEvent.Write", value))
	}
}

type FfiDestroyerSdkEvent struct{}

func (_ FfiDestroyerSdkEvent) Destroy(value SdkEvent) {
	value.Destroy()
}

type SendDestination interface {
	Destroy()
}
type SendDestinationLiquidAddress struct {
	AddressData   LiquidAddressData
	Bip353Address *string
}

func (e SendDestinationLiquidAddress) Destroy() {
	FfiDestroyerLiquidAddressData{}.Destroy(e.AddressData)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type SendDestinationBolt11 struct {
	Invoice       LnInvoice
	Bip353Address *string
}

func (e SendDestinationBolt11) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(e.Invoice)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type SendDestinationBolt12 struct {
	Offer             LnOffer
	ReceiverAmountSat uint64
	Bip353Address     *string
}

func (e SendDestinationBolt12) Destroy() {
	FfiDestroyerLnOffer{}.Destroy(e.Offer)
	FfiDestroyerUint64{}.Destroy(e.ReceiverAmountSat)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type FfiConverterSendDestination struct{}

var FfiConverterSendDestinationINSTANCE = FfiConverterSendDestination{}

func (c FfiConverterSendDestination) Lift(rb RustBufferI) SendDestination {
	return LiftFromRustBuffer[SendDestination](c, rb)
}

func (c FfiConverterSendDestination) Lower(value SendDestination) C.RustBuffer {
	return LowerIntoRustBuffer[SendDestination](c, value)
}
func (FfiConverterSendDestination) Read(reader io.Reader) SendDestination {
	id := readInt32(reader)
	switch id {
	case 1:
		return SendDestinationLiquidAddress{
			FfiConverterLiquidAddressDataINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 2:
		return SendDestinationBolt11{
			FfiConverterLnInvoiceINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 3:
		return SendDestinationBolt12{
			FfiConverterLnOfferINSTANCE.Read(reader),
			FfiConverterUint64INSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterSendDestination.Read()", id))
	}
}

func (FfiConverterSendDestination) Write(writer io.Writer, value SendDestination) {
	switch variant_value := value.(type) {
	case SendDestinationLiquidAddress:
		writeInt32(writer, 1)
		FfiConverterLiquidAddressDataINSTANCE.Write(writer, variant_value.AddressData)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case SendDestinationBolt11:
		writeInt32(writer, 2)
		FfiConverterLnInvoiceINSTANCE.Write(writer, variant_value.Invoice)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case SendDestinationBolt12:
		writeInt32(writer, 3)
		FfiConverterLnOfferINSTANCE.Write(writer, variant_value.Offer)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.ReceiverAmountSat)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterSendDestination.Write", value))
	}
}

type FfiDestroyerSendDestination struct{}

func (_ FfiDestroyerSendDestination) Destroy(value SendDestination) {
	value.Destroy()
}

type SignerError struct {
	err error
}

// Convience method to turn *SignerError into error
// Avoiding treating nil pointer as non nil error interface
func (err *SignerError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err SignerError) Error() string {
	return fmt.Sprintf("SignerError: %s", err.err.Error())
}

func (err SignerError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSignerErrorGeneric = fmt.Errorf("SignerErrorGeneric")

// Variant structs
type SignerErrorGeneric struct {
	Err string
}

func NewSignerErrorGeneric(
	err string,
) *SignerError {
	return &SignerError{err: &SignerErrorGeneric{
		Err: err}}
}

func (e SignerErrorGeneric) destroy() {
	FfiDestroyerString{}.Destroy(e.Err)
}

func (err SignerErrorGeneric) Error() string {
	return fmt.Sprint("Generic",
		": ",

		"Err=",
		err.Err,
	)
}

func (self SignerErrorGeneric) Is(target error) bool {
	return target == ErrSignerErrorGeneric
}

type FfiConverterSignerError struct{}

var FfiConverterSignerErrorINSTANCE = FfiConverterSignerError{}

func (c FfiConverterSignerError) Lift(eb RustBufferI) *SignerError {
	return LiftFromRustBuffer[*SignerError](c, eb)
}

func (c FfiConverterSignerError) Lower(value *SignerError) C.RustBuffer {
	return LowerIntoRustBuffer[*SignerError](c, value)
}

func (c FfiConverterSignerError) Read(reader io.Reader) *SignerError {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &SignerError{&SignerErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterSignerError.Read()", errorID))
	}
}

func (c FfiConverterSignerError) Write(writer io.Writer, value *SignerError) {
	switch variantValue := value.err.(type) {
	case *SignerErrorGeneric:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterSignerError.Write", value))
	}
}

type FfiDestroyerSignerError struct{}

func (_ FfiDestroyerSignerError) Destroy(value *SignerError) {
	switch variantValue := value.err.(type) {
	case SignerErrorGeneric:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerSignerError.Destroy", value))
	}
}

type SuccessAction interface {
	Destroy()
}
type SuccessActionAes struct {
	Data AesSuccessActionData
}

func (e SuccessActionAes) Destroy() {
	FfiDestroyerAesSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionMessage) Destroy() {
	FfiDestroyerMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionUrl) Destroy() {
	FfiDestroyerUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterSuccessAction struct{}

var FfiConverterSuccessActionINSTANCE = FfiConverterSuccessAction{}

func (c FfiConverterSuccessAction) Lift(rb RustBufferI) SuccessAction {
	return LiftFromRustBuffer[SuccessAction](c, rb)
}

func (c FfiConverterSuccessAction) Lower(value SuccessAction) C.RustBuffer {
	return LowerIntoRustBuffer[SuccessAction](c, value)
}
func (FfiConverterSuccessAction) Read(reader io.Reader) SuccessAction {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionAes{
			FfiConverterAesSuccessActionDataINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionMessage{
			FfiConverterMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionUrl{
			FfiConverterUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterSuccessAction.Read()", id))
	}
}

func (FfiConverterSuccessAction) Write(writer io.Writer, value SuccessAction) {
	switch variant_value := value.(type) {
	case SuccessActionAes:
		writeInt32(writer, 1)
		FfiConverterAesSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionMessage:
		writeInt32(writer, 2)
		FfiConverterMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionUrl:
		writeInt32(writer, 3)
		FfiConverterUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterSuccessAction.Write", value))
	}
}

type FfiDestroyerSuccessAction struct{}

func (_ FfiDestroyerSuccessAction) Destroy(value SuccessAction) {
	value.Destroy()
}

type SuccessActionProcessed interface {
	Destroy()
}
type SuccessActionProcessedAes struct {
	Result AesSuccessActionDataResult
}

func (e SuccessActionProcessedAes) Destroy() {
	FfiDestroyerAesSuccessActionDataResult{}.Destroy(e.Result)
}

type SuccessActionProcessedMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionProcessedMessage) Destroy() {
	FfiDestroyerMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionProcessedUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionProcessedUrl) Destroy() {
	FfiDestroyerUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterSuccessActionProcessed struct{}

var FfiConverterSuccessActionProcessedINSTANCE = FfiConverterSuccessActionProcessed{}

func (c FfiConverterSuccessActionProcessed) Lift(rb RustBufferI) SuccessActionProcessed {
	return LiftFromRustBuffer[SuccessActionProcessed](c, rb)
}

func (c FfiConverterSuccessActionProcessed) Lower(value SuccessActionProcessed) C.RustBuffer {
	return LowerIntoRustBuffer[SuccessActionProcessed](c, value)
}
func (FfiConverterSuccessActionProcessed) Read(reader io.Reader) SuccessActionProcessed {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionProcessedAes{
			FfiConverterAesSuccessActionDataResultINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionProcessedMessage{
			FfiConverterMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionProcessedUrl{
			FfiConverterUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterSuccessActionProcessed.Read()", id))
	}
}

func (FfiConverterSuccessActionProcessed) Write(writer io.Writer, value SuccessActionProcessed) {
	switch variant_value := value.(type) {
	case SuccessActionProcessedAes:
		writeInt32(writer, 1)
		FfiConverterAesSuccessActionDataResultINSTANCE.Write(writer, variant_value.Result)
	case SuccessActionProcessedMessage:
		writeInt32(writer, 2)
		FfiConverterMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionProcessedUrl:
		writeInt32(writer, 3)
		FfiConverterUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterSuccessActionProcessed.Write", value))
	}
}

type FfiDestroyerSuccessActionProcessed struct{}

func (_ FfiDestroyerSuccessActionProcessed) Destroy(value SuccessActionProcessed) {
	value.Destroy()
}

type EventListener interface {
	OnEvent(e SdkEvent)
}

type FfiConverterCallbackInterfaceEventListener struct {
	handleMap *concurrentHandleMap[EventListener]
}

var FfiConverterCallbackInterfaceEventListenerINSTANCE = FfiConverterCallbackInterfaceEventListener{
	handleMap: newConcurrentHandleMap[EventListener](),
}

func (c FfiConverterCallbackInterfaceEventListener) Lift(handle uint64) EventListener {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return val
}

func (c FfiConverterCallbackInterfaceEventListener) Read(reader io.Reader) EventListener {
	return c.Lift(readUint64(reader))
}

func (c FfiConverterCallbackInterfaceEventListener) Lower(value EventListener) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(value))
}

func (c FfiConverterCallbackInterfaceEventListener) Write(writer io.Writer, value EventListener) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type FfiDestroyerCallbackInterfaceEventListener struct{}

func (FfiDestroyerCallbackInterfaceEventListener) Destroy(value EventListener) {}

type uniffiCallbackResult C.int8_t

const (
	uniffiIdxCallbackFree               uniffiCallbackResult = 0
	uniffiCallbackResultSuccess         uniffiCallbackResult = 0
	uniffiCallbackResultError           uniffiCallbackResult = 1
	uniffiCallbackUnexpectedResultError uniffiCallbackResult = 2
	uniffiCallbackCancelled             uniffiCallbackResult = 3
)

type concurrentHandleMap[T any] struct {
	handles       map[uint64]T
	currentHandle uint64
	lock          sync.RWMutex
}

func newConcurrentHandleMap[T any]() *concurrentHandleMap[T] {
	return &concurrentHandleMap[T]{
		handles: map[uint64]T{},
	}
}

func (cm *concurrentHandleMap[T]) insert(obj T) uint64 {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.currentHandle = cm.currentHandle + 1
	cm.handles[cm.currentHandle] = obj
	return cm.currentHandle
}

func (cm *concurrentHandleMap[T]) remove(handle uint64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	delete(cm.handles, handle)
}

func (cm *concurrentHandleMap[T]) tryGet(handle uint64) (T, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	val, ok := cm.handles[handle]
	return val, ok
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0(uniffiHandle C.uint64_t, e C.RustBuffer, uniffiOutReturn *C.void, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceEventListenerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	uniffiObj.OnEvent(
		FfiConverterSdkEventINSTANCE.Lift(GoRustBuffer{
			inner: e,
		}),
	)

}

var UniffiVTableCallbackInterfaceEventListenerINSTANCE = C.UniffiVTableCallbackInterfaceEventListener{
	onEvent: (C.UniffiCallbackInterfaceEventListenerMethod0)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0),

	uniffiFree: (C.UniffiCallbackInterfaceFree)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerFree),
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerFree
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceEventListenerFree(handle C.uint64_t) {
	FfiConverterCallbackInterfaceEventListenerINSTANCE.handleMap.remove(uint64(handle))
}

func (c FfiConverterCallbackInterfaceEventListener) register() {
	C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_vtable_eventlistener(&UniffiVTableCallbackInterfaceEventListenerINSTANCE)
}

type Logger interface {
	Log(l LogEntry)
}

type FfiConverterCallbackInterfaceLogger struct {
	handleMap *concurrentHandleMap[Logger]
}

var FfiConverterCallbackInterfaceLoggerINSTANCE = FfiConverterCallbackInterfaceLogger{
	handleMap: newConcurrentHandleMap[Logger](),
}

func (c FfiConverterCallbackInterfaceLogger) Lift(handle uint64) Logger {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return val
}

func (c FfiConverterCallbackInterfaceLogger) Read(reader io.Reader) Logger {
	return c.Lift(readUint64(reader))
}

func (c FfiConverterCallbackInterfaceLogger) Lower(value Logger) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(value))
}

func (c FfiConverterCallbackInterfaceLogger) Write(writer io.Writer, value Logger) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type FfiDestroyerCallbackInterfaceLogger struct{}

func (FfiDestroyerCallbackInterfaceLogger) Destroy(value Logger) {}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerMethod0
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerMethod0(uniffiHandle C.uint64_t, l C.RustBuffer, uniffiOutReturn *C.void, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceLoggerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	uniffiObj.Log(
		FfiConverterLogEntryINSTANCE.Lift(GoRustBuffer{
			inner: l,
		}),
	)

}

var UniffiVTableCallbackInterfaceLoggerINSTANCE = C.UniffiVTableCallbackInterfaceLogger{
	log: (C.UniffiCallbackInterfaceLoggerMethod0)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerMethod0),

	uniffiFree: (C.UniffiCallbackInterfaceFree)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerFree),
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerFree
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceLoggerFree(handle C.uint64_t) {
	FfiConverterCallbackInterfaceLoggerINSTANCE.handleMap.remove(uint64(handle))
}

func (c FfiConverterCallbackInterfaceLogger) register() {
	C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_vtable_logger(&UniffiVTableCallbackInterfaceLoggerINSTANCE)
}

type Signer interface {
	Xpub() ([]uint8, *SignerError)

	DeriveXpub(derivationPath string) ([]uint8, *SignerError)

	SignEcdsa(msg []uint8, derivationPath string) ([]uint8, *SignerError)

	SignEcdsaRecoverable(msg []uint8) ([]uint8, *SignerError)

	Slip77MasterBlindingKey() ([]uint8, *SignerError)

	HmacSha256(msg []uint8, derivationPath string) ([]uint8, *SignerError)

	EciesEncrypt(msg []uint8) ([]uint8, *SignerError)

	EciesDecrypt(msg []uint8) ([]uint8, *SignerError)
}

type FfiConverterCallbackInterfaceSigner struct {
	handleMap *concurrentHandleMap[Signer]
}

var FfiConverterCallbackInterfaceSignerINSTANCE = FfiConverterCallbackInterfaceSigner{
	handleMap: newConcurrentHandleMap[Signer](),
}

func (c FfiConverterCallbackInterfaceSigner) Lift(handle uint64) Signer {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return val
}

func (c FfiConverterCallbackInterfaceSigner) Read(reader io.Reader) Signer {
	return c.Lift(readUint64(reader))
}

func (c FfiConverterCallbackInterfaceSigner) Lower(value Signer) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(value))
}

func (c FfiConverterCallbackInterfaceSigner) Write(writer io.Writer, value Signer) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type FfiDestroyerCallbackInterfaceSigner struct{}

func (FfiDestroyerCallbackInterfaceSigner) Destroy(value Signer) {}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod0
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod0(uniffiHandle C.uint64_t, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.Xpub()

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod1
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod1(uniffiHandle C.uint64_t, derivationPath C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.DeriveXpub(
			FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: derivationPath,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod2
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod2(uniffiHandle C.uint64_t, msg C.RustBuffer, derivationPath C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.SignEcdsa(
			FfiConverterSequenceUint8INSTANCE.Lift(GoRustBuffer{
				inner: msg,
			}),
			FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: derivationPath,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod3
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod3(uniffiHandle C.uint64_t, msg C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.SignEcdsaRecoverable(
			FfiConverterSequenceUint8INSTANCE.Lift(GoRustBuffer{
				inner: msg,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod4
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod4(uniffiHandle C.uint64_t, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.Slip77MasterBlindingKey()

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod5
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod5(uniffiHandle C.uint64_t, msg C.RustBuffer, derivationPath C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.HmacSha256(
			FfiConverterSequenceUint8INSTANCE.Lift(GoRustBuffer{
				inner: msg,
			}),
			FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: derivationPath,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod6
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod6(uniffiHandle C.uint64_t, msg C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.EciesEncrypt(
			FfiConverterSequenceUint8INSTANCE.Lift(GoRustBuffer{
				inner: msg,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod7
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod7(uniffiHandle C.uint64_t, msg C.RustBuffer, uniffiOutReturn *C.RustBuffer, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	res, err :=
		uniffiObj.EciesDecrypt(
			FfiConverterSequenceUint8INSTANCE.Lift(GoRustBuffer{
				inner: msg,
			}),
		)

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			*callStatus = C.RustCallStatus{
				code: C.int8_t(uniffiCallbackUnexpectedResultError),
			}
			return
		}

		*callStatus = C.RustCallStatus{
			code:     C.int8_t(uniffiCallbackResultError),
			errorBuf: FfiConverterSignerErrorINSTANCE.Lower(err),
		}
		return
	}

	*uniffiOutReturn = FfiConverterSequenceUint8INSTANCE.Lower(res)
}

var UniffiVTableCallbackInterfaceSignerINSTANCE = C.UniffiVTableCallbackInterfaceSigner{
	xpub:                    (C.UniffiCallbackInterfaceSignerMethod0)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod0),
	deriveXpub:              (C.UniffiCallbackInterfaceSignerMethod1)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod1),
	signEcdsa:               (C.UniffiCallbackInterfaceSignerMethod2)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod2),
	signEcdsaRecoverable:    (C.UniffiCallbackInterfaceSignerMethod3)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod3),
	slip77MasterBlindingKey: (C.UniffiCallbackInterfaceSignerMethod4)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod4),
	hmacSha256:              (C.UniffiCallbackInterfaceSignerMethod5)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod5),
	eciesEncrypt:            (C.UniffiCallbackInterfaceSignerMethod6)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod6),
	eciesDecrypt:            (C.UniffiCallbackInterfaceSignerMethod7)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerMethod7),

	uniffiFree: (C.UniffiCallbackInterfaceFree)(C.breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerFree),
}

//export breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerFree
func breez_sdk_liquid_bindings_cgo_dispatchCallbackInterfaceSignerFree(handle C.uint64_t) {
	FfiConverterCallbackInterfaceSignerINSTANCE.handleMap.remove(uint64(handle))
}

func (c FfiConverterCallbackInterfaceSigner) register() {
	C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_vtable_signer(&UniffiVTableCallbackInterfaceSignerINSTANCE)
}

type FfiConverterOptionalUint32 struct{}

var FfiConverterOptionalUint32INSTANCE = FfiConverterOptionalUint32{}

func (c FfiConverterOptionalUint32) Lift(rb RustBufferI) *uint32 {
	return LiftFromRustBuffer[*uint32](c, rb)
}

func (_ FfiConverterOptionalUint32) Read(reader io.Reader) *uint32 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint32INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint32) Lower(value *uint32) C.RustBuffer {
	return LowerIntoRustBuffer[*uint32](c, value)
}

func (_ FfiConverterOptionalUint32) Write(writer io.Writer, value *uint32) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint32INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint32 struct{}

func (_ FfiDestroyerOptionalUint32) Destroy(value *uint32) {
	if value != nil {
		FfiDestroyerUint32{}.Destroy(*value)
	}
}

type FfiConverterOptionalUint64 struct{}

var FfiConverterOptionalUint64INSTANCE = FfiConverterOptionalUint64{}

func (c FfiConverterOptionalUint64) Lift(rb RustBufferI) *uint64 {
	return LiftFromRustBuffer[*uint64](c, rb)
}

func (_ FfiConverterOptionalUint64) Read(reader io.Reader) *uint64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint64) Lower(value *uint64) C.RustBuffer {
	return LowerIntoRustBuffer[*uint64](c, value)
}

func (_ FfiConverterOptionalUint64) Write(writer io.Writer, value *uint64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint64 struct{}

func (_ FfiDestroyerOptionalUint64) Destroy(value *uint64) {
	if value != nil {
		FfiDestroyerUint64{}.Destroy(*value)
	}
}

type FfiConverterOptionalInt64 struct{}

var FfiConverterOptionalInt64INSTANCE = FfiConverterOptionalInt64{}

func (c FfiConverterOptionalInt64) Lift(rb RustBufferI) *int64 {
	return LiftFromRustBuffer[*int64](c, rb)
}

func (_ FfiConverterOptionalInt64) Read(reader io.Reader) *int64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterInt64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalInt64) Lower(value *int64) C.RustBuffer {
	return LowerIntoRustBuffer[*int64](c, value)
}

func (_ FfiConverterOptionalInt64) Write(writer io.Writer, value *int64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterInt64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalInt64 struct{}

func (_ FfiDestroyerOptionalInt64) Destroy(value *int64) {
	if value != nil {
		FfiDestroyerInt64{}.Destroy(*value)
	}
}

type FfiConverterOptionalFloat64 struct{}

var FfiConverterOptionalFloat64INSTANCE = FfiConverterOptionalFloat64{}

func (c FfiConverterOptionalFloat64) Lift(rb RustBufferI) *float64 {
	return LiftFromRustBuffer[*float64](c, rb)
}

func (_ FfiConverterOptionalFloat64) Read(reader io.Reader) *float64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterFloat64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalFloat64) Lower(value *float64) C.RustBuffer {
	return LowerIntoRustBuffer[*float64](c, value)
}

func (_ FfiConverterOptionalFloat64) Write(writer io.Writer, value *float64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterFloat64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalFloat64 struct{}

func (_ FfiDestroyerOptionalFloat64) Destroy(value *float64) {
	if value != nil {
		FfiDestroyerFloat64{}.Destroy(*value)
	}
}

type FfiConverterOptionalBool struct{}

var FfiConverterOptionalBoolINSTANCE = FfiConverterOptionalBool{}

func (c FfiConverterOptionalBool) Lift(rb RustBufferI) *bool {
	return LiftFromRustBuffer[*bool](c, rb)
}

func (_ FfiConverterOptionalBool) Read(reader io.Reader) *bool {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterBoolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalBool) Lower(value *bool) C.RustBuffer {
	return LowerIntoRustBuffer[*bool](c, value)
}

func (_ FfiConverterOptionalBool) Write(writer io.Writer, value *bool) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterBoolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalBool struct{}

func (_ FfiDestroyerOptionalBool) Destroy(value *bool) {
	if value != nil {
		FfiDestroyerBool{}.Destroy(*value)
	}
}

type FfiConverterOptionalString struct{}

var FfiConverterOptionalStringINSTANCE = FfiConverterOptionalString{}

func (c FfiConverterOptionalString) Lift(rb RustBufferI) *string {
	return LiftFromRustBuffer[*string](c, rb)
}

func (_ FfiConverterOptionalString) Read(reader io.Reader) *string {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterStringINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalString) Lower(value *string) C.RustBuffer {
	return LowerIntoRustBuffer[*string](c, value)
}

func (_ FfiConverterOptionalString) Write(writer io.Writer, value *string) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalString struct{}

func (_ FfiDestroyerOptionalString) Destroy(value *string) {
	if value != nil {
		FfiDestroyerString{}.Destroy(*value)
	}
}

type FfiConverterOptionalAssetInfo struct{}

var FfiConverterOptionalAssetInfoINSTANCE = FfiConverterOptionalAssetInfo{}

func (c FfiConverterOptionalAssetInfo) Lift(rb RustBufferI) *AssetInfo {
	return LiftFromRustBuffer[*AssetInfo](c, rb)
}

func (_ FfiConverterOptionalAssetInfo) Read(reader io.Reader) *AssetInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterAssetInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalAssetInfo) Lower(value *AssetInfo) C.RustBuffer {
	return LowerIntoRustBuffer[*AssetInfo](c, value)
}

func (_ FfiConverterOptionalAssetInfo) Write(writer io.Writer, value *AssetInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterAssetInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalAssetInfo struct{}

func (_ FfiDestroyerOptionalAssetInfo) Destroy(value *AssetInfo) {
	if value != nil {
		FfiDestroyerAssetInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalLnUrlInfo struct{}

var FfiConverterOptionalLnUrlInfoINSTANCE = FfiConverterOptionalLnUrlInfo{}

func (c FfiConverterOptionalLnUrlInfo) Lift(rb RustBufferI) *LnUrlInfo {
	return LiftFromRustBuffer[*LnUrlInfo](c, rb)
}

func (_ FfiConverterOptionalLnUrlInfo) Read(reader io.Reader) *LnUrlInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterLnUrlInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalLnUrlInfo) Lower(value *LnUrlInfo) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlInfo](c, value)
}

func (_ FfiConverterOptionalLnUrlInfo) Write(writer io.Writer, value *LnUrlInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterLnUrlInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalLnUrlInfo struct{}

func (_ FfiDestroyerOptionalLnUrlInfo) Destroy(value *LnUrlInfo) {
	if value != nil {
		FfiDestroyerLnUrlInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalPayment struct{}

var FfiConverterOptionalPaymentINSTANCE = FfiConverterOptionalPayment{}

func (c FfiConverterOptionalPayment) Lift(rb RustBufferI) *Payment {
	return LiftFromRustBuffer[*Payment](c, rb)
}

func (_ FfiConverterOptionalPayment) Read(reader io.Reader) *Payment {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterPaymentINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalPayment) Lower(value *Payment) C.RustBuffer {
	return LowerIntoRustBuffer[*Payment](c, value)
}

func (_ FfiConverterOptionalPayment) Write(writer io.Writer, value *Payment) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterPaymentINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalPayment struct{}

func (_ FfiDestroyerOptionalPayment) Destroy(value *Payment) {
	if value != nil {
		FfiDestroyerPayment{}.Destroy(*value)
	}
}

type FfiConverterOptionalSymbol struct{}

var FfiConverterOptionalSymbolINSTANCE = FfiConverterOptionalSymbol{}

func (c FfiConverterOptionalSymbol) Lift(rb RustBufferI) *Symbol {
	return LiftFromRustBuffer[*Symbol](c, rb)
}

func (_ FfiConverterOptionalSymbol) Read(reader io.Reader) *Symbol {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSymbolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSymbol) Lower(value *Symbol) C.RustBuffer {
	return LowerIntoRustBuffer[*Symbol](c, value)
}

func (_ FfiConverterOptionalSymbol) Write(writer io.Writer, value *Symbol) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSymbolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSymbol struct{}

func (_ FfiDestroyerOptionalSymbol) Destroy(value *Symbol) {
	if value != nil {
		FfiDestroyerSymbol{}.Destroy(*value)
	}
}

type FfiConverterOptionalAmount struct{}

var FfiConverterOptionalAmountINSTANCE = FfiConverterOptionalAmount{}

func (c FfiConverterOptionalAmount) Lift(rb RustBufferI) *Amount {
	return LiftFromRustBuffer[*Amount](c, rb)
}

func (_ FfiConverterOptionalAmount) Read(reader io.Reader) *Amount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalAmount) Lower(value *Amount) C.RustBuffer {
	return LowerIntoRustBuffer[*Amount](c, value)
}

func (_ FfiConverterOptionalAmount) Write(writer io.Writer, value *Amount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalAmount struct{}

func (_ FfiDestroyerOptionalAmount) Destroy(value *Amount) {
	if value != nil {
		FfiDestroyerAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalListPaymentDetails struct{}

var FfiConverterOptionalListPaymentDetailsINSTANCE = FfiConverterOptionalListPaymentDetails{}

func (c FfiConverterOptionalListPaymentDetails) Lift(rb RustBufferI) *ListPaymentDetails {
	return LiftFromRustBuffer[*ListPaymentDetails](c, rb)
}

func (_ FfiConverterOptionalListPaymentDetails) Read(reader io.Reader) *ListPaymentDetails {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterListPaymentDetailsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalListPaymentDetails) Lower(value *ListPaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[*ListPaymentDetails](c, value)
}

func (_ FfiConverterOptionalListPaymentDetails) Write(writer io.Writer, value *ListPaymentDetails) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterListPaymentDetailsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalListPaymentDetails struct{}

func (_ FfiDestroyerOptionalListPaymentDetails) Destroy(value *ListPaymentDetails) {
	if value != nil {
		FfiDestroyerListPaymentDetails{}.Destroy(*value)
	}
}

type FfiConverterOptionalPayAmount struct{}

var FfiConverterOptionalPayAmountINSTANCE = FfiConverterOptionalPayAmount{}

func (c FfiConverterOptionalPayAmount) Lift(rb RustBufferI) *PayAmount {
	return LiftFromRustBuffer[*PayAmount](c, rb)
}

func (_ FfiConverterOptionalPayAmount) Read(reader io.Reader) *PayAmount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterPayAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalPayAmount) Lower(value *PayAmount) C.RustBuffer {
	return LowerIntoRustBuffer[*PayAmount](c, value)
}

func (_ FfiConverterOptionalPayAmount) Write(writer io.Writer, value *PayAmount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterPayAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalPayAmount struct{}

func (_ FfiDestroyerOptionalPayAmount) Destroy(value *PayAmount) {
	if value != nil {
		FfiDestroyerPayAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalReceiveAmount struct{}

var FfiConverterOptionalReceiveAmountINSTANCE = FfiConverterOptionalReceiveAmount{}

func (c FfiConverterOptionalReceiveAmount) Lift(rb RustBufferI) *ReceiveAmount {
	return LiftFromRustBuffer[*ReceiveAmount](c, rb)
}

func (_ FfiConverterOptionalReceiveAmount) Read(reader io.Reader) *ReceiveAmount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterReceiveAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalReceiveAmount) Lower(value *ReceiveAmount) C.RustBuffer {
	return LowerIntoRustBuffer[*ReceiveAmount](c, value)
}

func (_ FfiConverterOptionalReceiveAmount) Write(writer io.Writer, value *ReceiveAmount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterReceiveAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalReceiveAmount struct{}

func (_ FfiDestroyerOptionalReceiveAmount) Destroy(value *ReceiveAmount) {
	if value != nil {
		FfiDestroyerReceiveAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalSuccessAction struct{}

var FfiConverterOptionalSuccessActionINSTANCE = FfiConverterOptionalSuccessAction{}

func (c FfiConverterOptionalSuccessAction) Lift(rb RustBufferI) *SuccessAction {
	return LiftFromRustBuffer[*SuccessAction](c, rb)
}

func (_ FfiConverterOptionalSuccessAction) Read(reader io.Reader) *SuccessAction {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSuccessActionINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSuccessAction) Lower(value *SuccessAction) C.RustBuffer {
	return LowerIntoRustBuffer[*SuccessAction](c, value)
}

func (_ FfiConverterOptionalSuccessAction) Write(writer io.Writer, value *SuccessAction) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSuccessActionINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSuccessAction struct{}

func (_ FfiDestroyerOptionalSuccessAction) Destroy(value *SuccessAction) {
	if value != nil {
		FfiDestroyerSuccessAction{}.Destroy(*value)
	}
}

type FfiConverterOptionalSuccessActionProcessed struct{}

var FfiConverterOptionalSuccessActionProcessedINSTANCE = FfiConverterOptionalSuccessActionProcessed{}

func (c FfiConverterOptionalSuccessActionProcessed) Lift(rb RustBufferI) *SuccessActionProcessed {
	return LiftFromRustBuffer[*SuccessActionProcessed](c, rb)
}

func (_ FfiConverterOptionalSuccessActionProcessed) Read(reader io.Reader) *SuccessActionProcessed {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSuccessActionProcessedINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSuccessActionProcessed) Lower(value *SuccessActionProcessed) C.RustBuffer {
	return LowerIntoRustBuffer[*SuccessActionProcessed](c, value)
}

func (_ FfiConverterOptionalSuccessActionProcessed) Write(writer io.Writer, value *SuccessActionProcessed) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSuccessActionProcessedINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSuccessActionProcessed struct{}

func (_ FfiDestroyerOptionalSuccessActionProcessed) Destroy(value *SuccessActionProcessed) {
	if value != nil {
		FfiDestroyerSuccessActionProcessed{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceUint8 struct{}

var FfiConverterOptionalSequenceUint8INSTANCE = FfiConverterOptionalSequenceUint8{}

func (c FfiConverterOptionalSequenceUint8) Lift(rb RustBufferI) *[]uint8 {
	return LiftFromRustBuffer[*[]uint8](c, rb)
}

func (_ FfiConverterOptionalSequenceUint8) Read(reader io.Reader) *[]uint8 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceUint8INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceUint8) Lower(value *[]uint8) C.RustBuffer {
	return LowerIntoRustBuffer[*[]uint8](c, value)
}

func (_ FfiConverterOptionalSequenceUint8) Write(writer io.Writer, value *[]uint8) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceUint8INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceUint8 struct{}

func (_ FfiDestroyerOptionalSequenceUint8) Destroy(value *[]uint8) {
	if value != nil {
		FfiDestroyerSequenceUint8{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceAssetMetadata struct{}

var FfiConverterOptionalSequenceAssetMetadataINSTANCE = FfiConverterOptionalSequenceAssetMetadata{}

func (c FfiConverterOptionalSequenceAssetMetadata) Lift(rb RustBufferI) *[]AssetMetadata {
	return LiftFromRustBuffer[*[]AssetMetadata](c, rb)
}

func (_ FfiConverterOptionalSequenceAssetMetadata) Read(reader io.Reader) *[]AssetMetadata {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceAssetMetadataINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceAssetMetadata) Lower(value *[]AssetMetadata) C.RustBuffer {
	return LowerIntoRustBuffer[*[]AssetMetadata](c, value)
}

func (_ FfiConverterOptionalSequenceAssetMetadata) Write(writer io.Writer, value *[]AssetMetadata) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceAssetMetadataINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceAssetMetadata struct{}

func (_ FfiDestroyerOptionalSequenceAssetMetadata) Destroy(value *[]AssetMetadata) {
	if value != nil {
		FfiDestroyerSequenceAssetMetadata{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceExternalInputParser struct{}

var FfiConverterOptionalSequenceExternalInputParserINSTANCE = FfiConverterOptionalSequenceExternalInputParser{}

func (c FfiConverterOptionalSequenceExternalInputParser) Lift(rb RustBufferI) *[]ExternalInputParser {
	return LiftFromRustBuffer[*[]ExternalInputParser](c, rb)
}

func (_ FfiConverterOptionalSequenceExternalInputParser) Read(reader io.Reader) *[]ExternalInputParser {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceExternalInputParserINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceExternalInputParser) Lower(value *[]ExternalInputParser) C.RustBuffer {
	return LowerIntoRustBuffer[*[]ExternalInputParser](c, value)
}

func (_ FfiConverterOptionalSequenceExternalInputParser) Write(writer io.Writer, value *[]ExternalInputParser) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceExternalInputParserINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceExternalInputParser struct{}

func (_ FfiDestroyerOptionalSequenceExternalInputParser) Destroy(value *[]ExternalInputParser) {
	if value != nil {
		FfiDestroyerSequenceExternalInputParser{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequencePaymentState struct{}

var FfiConverterOptionalSequencePaymentStateINSTANCE = FfiConverterOptionalSequencePaymentState{}

func (c FfiConverterOptionalSequencePaymentState) Lift(rb RustBufferI) *[]PaymentState {
	return LiftFromRustBuffer[*[]PaymentState](c, rb)
}

func (_ FfiConverterOptionalSequencePaymentState) Read(reader io.Reader) *[]PaymentState {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequencePaymentStateINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequencePaymentState) Lower(value *[]PaymentState) C.RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentState](c, value)
}

func (_ FfiConverterOptionalSequencePaymentState) Write(writer io.Writer, value *[]PaymentState) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequencePaymentStateINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequencePaymentState struct{}

func (_ FfiDestroyerOptionalSequencePaymentState) Destroy(value *[]PaymentState) {
	if value != nil {
		FfiDestroyerSequencePaymentState{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequencePaymentType struct{}

var FfiConverterOptionalSequencePaymentTypeINSTANCE = FfiConverterOptionalSequencePaymentType{}

func (c FfiConverterOptionalSequencePaymentType) Lift(rb RustBufferI) *[]PaymentType {
	return LiftFromRustBuffer[*[]PaymentType](c, rb)
}

func (_ FfiConverterOptionalSequencePaymentType) Read(reader io.Reader) *[]PaymentType {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequencePaymentTypeINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequencePaymentType) Lower(value *[]PaymentType) C.RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentType](c, value)
}

func (_ FfiConverterOptionalSequencePaymentType) Write(writer io.Writer, value *[]PaymentType) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequencePaymentTypeINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequencePaymentType struct{}

func (_ FfiDestroyerOptionalSequencePaymentType) Destroy(value *[]PaymentType) {
	if value != nil {
		FfiDestroyerSequencePaymentType{}.Destroy(*value)
	}
}

type FfiConverterSequenceUint8 struct{}

var FfiConverterSequenceUint8INSTANCE = FfiConverterSequenceUint8{}

func (c FfiConverterSequenceUint8) Lift(rb RustBufferI) []uint8 {
	return LiftFromRustBuffer[[]uint8](c, rb)
}

func (c FfiConverterSequenceUint8) Read(reader io.Reader) []uint8 {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]uint8, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterUint8INSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceUint8) Lower(value []uint8) C.RustBuffer {
	return LowerIntoRustBuffer[[]uint8](c, value)
}

func (c FfiConverterSequenceUint8) Write(writer io.Writer, value []uint8) {
	if len(value) > math.MaxInt32 {
		panic("[]uint8 is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterUint8INSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceUint8 struct{}

func (FfiDestroyerSequenceUint8) Destroy(sequence []uint8) {
	for _, value := range sequence {
		FfiDestroyerUint8{}.Destroy(value)
	}
}

type FfiConverterSequenceString struct{}

var FfiConverterSequenceStringINSTANCE = FfiConverterSequenceString{}

func (c FfiConverterSequenceString) Lift(rb RustBufferI) []string {
	return LiftFromRustBuffer[[]string](c, rb)
}

func (c FfiConverterSequenceString) Read(reader io.Reader) []string {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]string, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterStringINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceString) Lower(value []string) C.RustBuffer {
	return LowerIntoRustBuffer[[]string](c, value)
}

func (c FfiConverterSequenceString) Write(writer io.Writer, value []string) {
	if len(value) > math.MaxInt32 {
		panic("[]string is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterStringINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceString struct{}

func (FfiDestroyerSequenceString) Destroy(sequence []string) {
	for _, value := range sequence {
		FfiDestroyerString{}.Destroy(value)
	}
}

type FfiConverterSequenceAssetBalance struct{}

var FfiConverterSequenceAssetBalanceINSTANCE = FfiConverterSequenceAssetBalance{}

func (c FfiConverterSequenceAssetBalance) Lift(rb RustBufferI) []AssetBalance {
	return LiftFromRustBuffer[[]AssetBalance](c, rb)
}

func (c FfiConverterSequenceAssetBalance) Read(reader io.Reader) []AssetBalance {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]AssetBalance, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterAssetBalanceINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceAssetBalance) Lower(value []AssetBalance) C.RustBuffer {
	return LowerIntoRustBuffer[[]AssetBalance](c, value)
}

func (c FfiConverterSequenceAssetBalance) Write(writer io.Writer, value []AssetBalance) {
	if len(value) > math.MaxInt32 {
		panic("[]AssetBalance is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterAssetBalanceINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceAssetBalance struct{}

func (FfiDestroyerSequenceAssetBalance) Destroy(sequence []AssetBalance) {
	for _, value := range sequence {
		FfiDestroyerAssetBalance{}.Destroy(value)
	}
}

type FfiConverterSequenceAssetMetadata struct{}

var FfiConverterSequenceAssetMetadataINSTANCE = FfiConverterSequenceAssetMetadata{}

func (c FfiConverterSequenceAssetMetadata) Lift(rb RustBufferI) []AssetMetadata {
	return LiftFromRustBuffer[[]AssetMetadata](c, rb)
}

func (c FfiConverterSequenceAssetMetadata) Read(reader io.Reader) []AssetMetadata {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]AssetMetadata, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterAssetMetadataINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceAssetMetadata) Lower(value []AssetMetadata) C.RustBuffer {
	return LowerIntoRustBuffer[[]AssetMetadata](c, value)
}

func (c FfiConverterSequenceAssetMetadata) Write(writer io.Writer, value []AssetMetadata) {
	if len(value) > math.MaxInt32 {
		panic("[]AssetMetadata is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterAssetMetadataINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceAssetMetadata struct{}

func (FfiDestroyerSequenceAssetMetadata) Destroy(sequence []AssetMetadata) {
	for _, value := range sequence {
		FfiDestroyerAssetMetadata{}.Destroy(value)
	}
}

type FfiConverterSequenceExternalInputParser struct{}

var FfiConverterSequenceExternalInputParserINSTANCE = FfiConverterSequenceExternalInputParser{}

func (c FfiConverterSequenceExternalInputParser) Lift(rb RustBufferI) []ExternalInputParser {
	return LiftFromRustBuffer[[]ExternalInputParser](c, rb)
}

func (c FfiConverterSequenceExternalInputParser) Read(reader io.Reader) []ExternalInputParser {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ExternalInputParser, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterExternalInputParserINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceExternalInputParser) Lower(value []ExternalInputParser) C.RustBuffer {
	return LowerIntoRustBuffer[[]ExternalInputParser](c, value)
}

func (c FfiConverterSequenceExternalInputParser) Write(writer io.Writer, value []ExternalInputParser) {
	if len(value) > math.MaxInt32 {
		panic("[]ExternalInputParser is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterExternalInputParserINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceExternalInputParser struct{}

func (FfiDestroyerSequenceExternalInputParser) Destroy(sequence []ExternalInputParser) {
	for _, value := range sequence {
		FfiDestroyerExternalInputParser{}.Destroy(value)
	}
}

type FfiConverterSequenceFiatCurrency struct{}

var FfiConverterSequenceFiatCurrencyINSTANCE = FfiConverterSequenceFiatCurrency{}

func (c FfiConverterSequenceFiatCurrency) Lift(rb RustBufferI) []FiatCurrency {
	return LiftFromRustBuffer[[]FiatCurrency](c, rb)
}

func (c FfiConverterSequenceFiatCurrency) Read(reader io.Reader) []FiatCurrency {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]FiatCurrency, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterFiatCurrencyINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceFiatCurrency) Lower(value []FiatCurrency) C.RustBuffer {
	return LowerIntoRustBuffer[[]FiatCurrency](c, value)
}

func (c FfiConverterSequenceFiatCurrency) Write(writer io.Writer, value []FiatCurrency) {
	if len(value) > math.MaxInt32 {
		panic("[]FiatCurrency is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterFiatCurrencyINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceFiatCurrency struct{}

func (FfiDestroyerSequenceFiatCurrency) Destroy(sequence []FiatCurrency) {
	for _, value := range sequence {
		FfiDestroyerFiatCurrency{}.Destroy(value)
	}
}

type FfiConverterSequenceLnOfferBlindedPath struct{}

var FfiConverterSequenceLnOfferBlindedPathINSTANCE = FfiConverterSequenceLnOfferBlindedPath{}

func (c FfiConverterSequenceLnOfferBlindedPath) Lift(rb RustBufferI) []LnOfferBlindedPath {
	return LiftFromRustBuffer[[]LnOfferBlindedPath](c, rb)
}

func (c FfiConverterSequenceLnOfferBlindedPath) Read(reader io.Reader) []LnOfferBlindedPath {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LnOfferBlindedPath, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLnOfferBlindedPathINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLnOfferBlindedPath) Lower(value []LnOfferBlindedPath) C.RustBuffer {
	return LowerIntoRustBuffer[[]LnOfferBlindedPath](c, value)
}

func (c FfiConverterSequenceLnOfferBlindedPath) Write(writer io.Writer, value []LnOfferBlindedPath) {
	if len(value) > math.MaxInt32 {
		panic("[]LnOfferBlindedPath is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLnOfferBlindedPathINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLnOfferBlindedPath struct{}

func (FfiDestroyerSequenceLnOfferBlindedPath) Destroy(sequence []LnOfferBlindedPath) {
	for _, value := range sequence {
		FfiDestroyerLnOfferBlindedPath{}.Destroy(value)
	}
}

type FfiConverterSequenceLocaleOverrides struct{}

var FfiConverterSequenceLocaleOverridesINSTANCE = FfiConverterSequenceLocaleOverrides{}

func (c FfiConverterSequenceLocaleOverrides) Lift(rb RustBufferI) []LocaleOverrides {
	return LiftFromRustBuffer[[]LocaleOverrides](c, rb)
}

func (c FfiConverterSequenceLocaleOverrides) Read(reader io.Reader) []LocaleOverrides {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocaleOverrides, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLocaleOverridesINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLocaleOverrides) Lower(value []LocaleOverrides) C.RustBuffer {
	return LowerIntoRustBuffer[[]LocaleOverrides](c, value)
}

func (c FfiConverterSequenceLocaleOverrides) Write(writer io.Writer, value []LocaleOverrides) {
	if len(value) > math.MaxInt32 {
		panic("[]LocaleOverrides is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLocaleOverridesINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLocaleOverrides struct{}

func (FfiDestroyerSequenceLocaleOverrides) Destroy(sequence []LocaleOverrides) {
	for _, value := range sequence {
		FfiDestroyerLocaleOverrides{}.Destroy(value)
	}
}

type FfiConverterSequenceLocalizedName struct{}

var FfiConverterSequenceLocalizedNameINSTANCE = FfiConverterSequenceLocalizedName{}

func (c FfiConverterSequenceLocalizedName) Lift(rb RustBufferI) []LocalizedName {
	return LiftFromRustBuffer[[]LocalizedName](c, rb)
}

func (c FfiConverterSequenceLocalizedName) Read(reader io.Reader) []LocalizedName {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocalizedName, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLocalizedNameINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLocalizedName) Lower(value []LocalizedName) C.RustBuffer {
	return LowerIntoRustBuffer[[]LocalizedName](c, value)
}

func (c FfiConverterSequenceLocalizedName) Write(writer io.Writer, value []LocalizedName) {
	if len(value) > math.MaxInt32 {
		panic("[]LocalizedName is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLocalizedNameINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLocalizedName struct{}

func (FfiDestroyerSequenceLocalizedName) Destroy(sequence []LocalizedName) {
	for _, value := range sequence {
		FfiDestroyerLocalizedName{}.Destroy(value)
	}
}

type FfiConverterSequencePayment struct{}

var FfiConverterSequencePaymentINSTANCE = FfiConverterSequencePayment{}

func (c FfiConverterSequencePayment) Lift(rb RustBufferI) []Payment {
	return LiftFromRustBuffer[[]Payment](c, rb)
}

func (c FfiConverterSequencePayment) Read(reader io.Reader) []Payment {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Payment, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterPaymentINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequencePayment) Lower(value []Payment) C.RustBuffer {
	return LowerIntoRustBuffer[[]Payment](c, value)
}

func (c FfiConverterSequencePayment) Write(writer io.Writer, value []Payment) {
	if len(value) > math.MaxInt32 {
		panic("[]Payment is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterPaymentINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequencePayment struct{}

func (FfiDestroyerSequencePayment) Destroy(sequence []Payment) {
	for _, value := range sequence {
		FfiDestroyerPayment{}.Destroy(value)
	}
}

type FfiConverterSequenceRate struct{}

var FfiConverterSequenceRateINSTANCE = FfiConverterSequenceRate{}

func (c FfiConverterSequenceRate) Lift(rb RustBufferI) []Rate {
	return LiftFromRustBuffer[[]Rate](c, rb)
}

func (c FfiConverterSequenceRate) Read(reader io.Reader) []Rate {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Rate, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRate) Lower(value []Rate) C.RustBuffer {
	return LowerIntoRustBuffer[[]Rate](c, value)
}

func (c FfiConverterSequenceRate) Write(writer io.Writer, value []Rate) {
	if len(value) > math.MaxInt32 {
		panic("[]Rate is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRate struct{}

func (FfiDestroyerSequenceRate) Destroy(sequence []Rate) {
	for _, value := range sequence {
		FfiDestroyerRate{}.Destroy(value)
	}
}

type FfiConverterSequenceRefundableSwap struct{}

var FfiConverterSequenceRefundableSwapINSTANCE = FfiConverterSequenceRefundableSwap{}

func (c FfiConverterSequenceRefundableSwap) Lift(rb RustBufferI) []RefundableSwap {
	return LiftFromRustBuffer[[]RefundableSwap](c, rb)
}

func (c FfiConverterSequenceRefundableSwap) Read(reader io.Reader) []RefundableSwap {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RefundableSwap, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRefundableSwapINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRefundableSwap) Lower(value []RefundableSwap) C.RustBuffer {
	return LowerIntoRustBuffer[[]RefundableSwap](c, value)
}

func (c FfiConverterSequenceRefundableSwap) Write(writer io.Writer, value []RefundableSwap) {
	if len(value) > math.MaxInt32 {
		panic("[]RefundableSwap is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRefundableSwapINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRefundableSwap struct{}

func (FfiDestroyerSequenceRefundableSwap) Destroy(sequence []RefundableSwap) {
	for _, value := range sequence {
		FfiDestroyerRefundableSwap{}.Destroy(value)
	}
}

type FfiConverterSequenceRouteHint struct{}

var FfiConverterSequenceRouteHintINSTANCE = FfiConverterSequenceRouteHint{}

func (c FfiConverterSequenceRouteHint) Lift(rb RustBufferI) []RouteHint {
	return LiftFromRustBuffer[[]RouteHint](c, rb)
}

func (c FfiConverterSequenceRouteHint) Read(reader io.Reader) []RouteHint {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHint, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRouteHintINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRouteHint) Lower(value []RouteHint) C.RustBuffer {
	return LowerIntoRustBuffer[[]RouteHint](c, value)
}

func (c FfiConverterSequenceRouteHint) Write(writer io.Writer, value []RouteHint) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHint is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRouteHintINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRouteHint struct{}

func (FfiDestroyerSequenceRouteHint) Destroy(sequence []RouteHint) {
	for _, value := range sequence {
		FfiDestroyerRouteHint{}.Destroy(value)
	}
}

type FfiConverterSequenceRouteHintHop struct{}

var FfiConverterSequenceRouteHintHopINSTANCE = FfiConverterSequenceRouteHintHop{}

func (c FfiConverterSequenceRouteHintHop) Lift(rb RustBufferI) []RouteHintHop {
	return LiftFromRustBuffer[[]RouteHintHop](c, rb)
}

func (c FfiConverterSequenceRouteHintHop) Read(reader io.Reader) []RouteHintHop {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHintHop, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRouteHintHopINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRouteHintHop) Lower(value []RouteHintHop) C.RustBuffer {
	return LowerIntoRustBuffer[[]RouteHintHop](c, value)
}

func (c FfiConverterSequenceRouteHintHop) Write(writer io.Writer, value []RouteHintHop) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHintHop is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRouteHintHopINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRouteHintHop struct{}

func (FfiDestroyerSequenceRouteHintHop) Destroy(sequence []RouteHintHop) {
	for _, value := range sequence {
		FfiDestroyerRouteHintHop{}.Destroy(value)
	}
}

type FfiConverterSequencePaymentState struct{}

var FfiConverterSequencePaymentStateINSTANCE = FfiConverterSequencePaymentState{}

func (c FfiConverterSequencePaymentState) Lift(rb RustBufferI) []PaymentState {
	return LiftFromRustBuffer[[]PaymentState](c, rb)
}

func (c FfiConverterSequencePaymentState) Read(reader io.Reader) []PaymentState {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentState, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterPaymentStateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequencePaymentState) Lower(value []PaymentState) C.RustBuffer {
	return LowerIntoRustBuffer[[]PaymentState](c, value)
}

func (c FfiConverterSequencePaymentState) Write(writer io.Writer, value []PaymentState) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentState is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterPaymentStateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequencePaymentState struct{}

func (FfiDestroyerSequencePaymentState) Destroy(sequence []PaymentState) {
	for _, value := range sequence {
		FfiDestroyerPaymentState{}.Destroy(value)
	}
}

type FfiConverterSequencePaymentType struct{}

var FfiConverterSequencePaymentTypeINSTANCE = FfiConverterSequencePaymentType{}

func (c FfiConverterSequencePaymentType) Lift(rb RustBufferI) []PaymentType {
	return LiftFromRustBuffer[[]PaymentType](c, rb)
}

func (c FfiConverterSequencePaymentType) Read(reader io.Reader) []PaymentType {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentType, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterPaymentTypeINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequencePaymentType) Lower(value []PaymentType) C.RustBuffer {
	return LowerIntoRustBuffer[[]PaymentType](c, value)
}

func (c FfiConverterSequencePaymentType) Write(writer io.Writer, value []PaymentType) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentType is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterPaymentTypeINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequencePaymentType struct{}

func (FfiDestroyerSequencePaymentType) Destroy(sequence []PaymentType) {
	for _, value := range sequence {
		FfiDestroyerPaymentType{}.Destroy(value)
	}
}

func Connect(req ConnectRequest) (*BindingLiquidSdk, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_connect(FfiConverterConnectRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BindingLiquidSdk
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBindingLiquidSdkINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ConnectWithSigner(req ConnectWithSignerRequest, signer Signer) (*BindingLiquidSdk, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_connect_with_signer(FfiConverterConnectWithSignerRequestINSTANCE.Lower(req), FfiConverterCallbackInterfaceSignerINSTANCE.Lower(signer), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BindingLiquidSdk
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBindingLiquidSdkINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func DefaultConfig(network LiquidNetwork, breezApiKey *string) (Config, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_func_default_config(FfiConverterLiquidNetworkINSTANCE.Lower(network), FfiConverterOptionalStringINSTANCE.Lower(breezApiKey), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue Config
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterConfigINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInvoice(input string) (LnInvoice, *PaymentError) {
	_uniffiRV, _uniffiErr := rustCallWithError[PaymentError](FfiConverterPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_liquid_bindings_fn_func_parse_invoice(FfiConverterStringINSTANCE.Lower(input), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnInvoice
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnInvoiceINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func SetLogger(logger Logger) *SdkError {
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_func_set_logger(FfiConverterCallbackInterfaceLoggerINSTANCE.Lower(logger), _uniffiStatus)
		return false
	})
	return _uniffiErr
}
