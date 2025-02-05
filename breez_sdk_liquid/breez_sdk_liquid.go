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

type RustBuffer = C.RustBuffer

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() int
	Capacity() int
}

func RustBufferFromExternal(b RustBufferI) RustBuffer {
	return RustBuffer{
		capacity: C.int(b.Capacity()),
		len:      C.int(b.Len()),
		data:     (*C.uchar)(b.Data()),
	}
}

func (cb RustBuffer) Capacity() int {
	return int(cb.capacity)
}

func (cb RustBuffer) Len() int {
	return int(cb.len)
}

func (cb RustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.data)
}

func (cb RustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.data), C.int(cb.len))
	return bytes.NewReader(b)
}

func (cb RustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_breez_sdk_liquid_bindings_rustbuffer_free(cb, status)
		return false
	})
}

func (cb RustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.data), C.int(cb.len))
}

func stringToRustBuffer(str string) RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) RustBuffer {
	if len(b) == 0 {
		return RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) RustBuffer {
		return C.ffi_breez_sdk_liquid_bindings_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) RustBuffer
}

type FfiConverter[GoType any, FfiType any] interface {
	Lift(value FfiType) GoType
	Lower(value GoType) FfiType
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

type FfiRustBufConverter[GoType any, FfiType any] interface {
	FfiConverter[GoType, FfiType]
	BufReader[GoType]
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) RustBuffer {
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

func rustCallWithError[U any](converter BufLifter[error], callback func(*C.RustCallStatus) U) (U, error) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)

	return returnValue, err
}

func checkCallStatus(converter BufLifter[error], status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		return converter.Lift(status.errorBuf)
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError(nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
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

	(&FfiConverterCallbackInterfaceEventListener{}).register()
	(&FfiConverterCallbackInterfaceLogger{}).register()
	(&FfiConverterCallbackInterfaceSigner{}).register()
	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 24
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_breez_sdk_liquid_bindings_uniffi_contract_version(uniffiStatus)
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("breez_sdk_liquid: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_connect(uniffiStatus)
		})
		if checksum != 31419 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_connect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_connect_with_signer(uniffiStatus)
		})
		if checksum != 56336 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_connect_with_signer: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_default_config(uniffiStatus)
		})
		if checksum != 28024 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_default_config: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_parse_invoice(uniffiStatus)
		})
		if checksum != 1802 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_parse_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_func_set_logger(uniffiStatus)
		})
		if checksum != 16687 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_func_set_logger: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_accept_payment_proposed_fees(uniffiStatus)
		})
		if checksum != 8720 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_accept_payment_proposed_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_add_event_listener(uniffiStatus)
		})
		if checksum != 16594 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_add_event_listener: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_backup(uniffiStatus)
		})
		if checksum != 65506 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_buy_bitcoin(uniffiStatus)
		})
		if checksum != 58975 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_check_message(uniffiStatus)
		})
		if checksum != 43888 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_check_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_disconnect(uniffiStatus)
		})
		if checksum != 23272 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_disconnect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_fiat_rates(uniffiStatus)
		})
		if checksum != 3014 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_fiat_rates: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_lightning_limits(uniffiStatus)
		})
		if checksum != 63386 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_lightning_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_onchain_limits(uniffiStatus)
		})
		if checksum != 59115 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_onchain_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_payment_proposed_fees(uniffiStatus)
		})
		if checksum != 23582 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_fetch_payment_proposed_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_info(uniffiStatus)
		})
		if checksum != 57337 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_payment(uniffiStatus)
		})
		if checksum != 55428 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_get_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_fiat_currencies(uniffiStatus)
		})
		if checksum != 10674 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_fiat_currencies: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_payments(uniffiStatus)
		})
		if checksum != 43948 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_refundables(uniffiStatus)
		})
		if checksum != 3276 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_list_refundables: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_auth(uniffiStatus)
		})
		if checksum != 5662 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_auth: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_pay(uniffiStatus)
		})
		if checksum != 37490 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_pay: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_withdraw(uniffiStatus)
		})
		if checksum != 45238 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_lnurl_withdraw: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_parse(uniffiStatus)
		})
		if checksum != 65160 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_parse: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_pay_onchain(uniffiStatus)
		})
		if checksum != 36501 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_buy_bitcoin(uniffiStatus)
		})
		if checksum != 3072 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_lnurl_pay(uniffiStatus)
		})
		if checksum != 62389 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_lnurl_pay: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_pay_onchain(uniffiStatus)
		})
		if checksum != 45645 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment(uniffiStatus)
		})
		if checksum != 7195 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_refund(uniffiStatus)
		})
		if checksum != 39396 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_send_payment(uniffiStatus)
		})
		if checksum != 17670 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_prepare_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_receive_payment(uniffiStatus)
		})
		if checksum != 24170 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_recommended_fees(uniffiStatus)
		})
		if checksum != 56530 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_recommended_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_refund(uniffiStatus)
		})
		if checksum != 8960 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_register_webhook(uniffiStatus)
		})
		if checksum != 48160 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_register_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_remove_event_listener(uniffiStatus)
		})
		if checksum != 42027 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_remove_event_listener: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_rescan_onchain_swaps(uniffiStatus)
		})
		if checksum != 21498 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_rescan_onchain_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_restore(uniffiStatus)
		})
		if checksum != 12644 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_restore: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_send_payment(uniffiStatus)
		})
		if checksum != 23038 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sign_message(uniffiStatus)
		})
		if checksum != 28732 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sign_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sync(uniffiStatus)
		})
		if checksum != 63769 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_sync: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_unregister_webhook(uniffiStatus)
		})
		if checksum != 49665 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_bindingliquidsdk_unregister_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_eventlistener_on_event(uniffiStatus)
		})
		if checksum != 62143 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_eventlistener_on_event: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_logger_log(uniffiStatus)
		})
		if checksum != 54784 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_logger_log: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_xpub(uniffiStatus)
		})
		if checksum != 39767 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_xpub: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_derive_xpub(uniffiStatus)
		})
		if checksum != 59515 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_derive_xpub: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa(uniffiStatus)
		})
		if checksum != 21427 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa_recoverable(uniffiStatus)
		})
		if checksum != 9552 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_sign_ecdsa_recoverable: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_slip77_master_blinding_key(uniffiStatus)
		})
		if checksum != 56356 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_slip77_master_blinding_key: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_hmac_sha256(uniffiStatus)
		})
		if checksum != 52627 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_hmac_sha256: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_encrypt(uniffiStatus)
		})
		if checksum != 8960 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk_liquid: uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_encrypt: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_liquid_bindings_checksum_method_signer_ecies_decrypt(uniffiStatus)
		})
		if checksum != 32966 {
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

func (FfiConverterString) Lower(value string) RustBuffer {
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
	pointer      unsafe.Pointer
	callCounter  atomic.Int64
	freeFunction func(unsafe.Pointer, *C.RustCallStatus)
	destroyed    atomic.Bool
}

func newFfiObject(pointer unsafe.Pointer, freeFunction func(unsafe.Pointer, *C.RustCallStatus)) FfiObject {
	return FfiObject{
		pointer:      pointer,
		freeFunction: freeFunction,
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

	return ffiObject.pointer
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

type BindingLiquidSdk struct {
	ffiObject FfiObject
}

func (_self *BindingLiquidSdk) AcceptPaymentProposedFees(req AcceptPaymentProposedFeesRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_accept_payment_proposed_fees(
			_pointer, FfiConverterTypeAcceptPaymentProposedFeesRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) AddEventListener(listener EventListener) (string, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_add_event_listener(
			_pointer, FfiConverterCallbackInterfaceEventListenerINSTANCE.Lower(listener), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Backup(req BackupRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_backup(
			_pointer, FfiConverterTypeBackupRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) BuyBitcoin(req BuyBitcoinRequest) (string, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_buy_bitcoin(
			_pointer, FfiConverterTypeBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) CheckMessage(req CheckMessageRequest) (CheckMessageResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_check_message(
			_pointer, FfiConverterTypeCheckMessageRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue CheckMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeCheckMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Disconnect() error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_disconnect(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) FetchFiatRates() ([]Rate, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_fiat_rates(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Rate
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeRateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchLightningLimits() (LightningPaymentLimitsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_lightning_limits(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LightningPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLightningPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchOnchainLimits() (OnchainPaymentLimitsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_onchain_limits(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OnchainPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeOnchainPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) FetchPaymentProposedFees(req FetchPaymentProposedFeesRequest) (FetchPaymentProposedFeesResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_fetch_payment_proposed_fees(
			_pointer, FfiConverterTypeFetchPaymentProposedFeesRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue FetchPaymentProposedFeesResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeFetchPaymentProposedFeesResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) GetInfo() (GetInfoResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_get_info(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue GetInfoResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeGetInfoResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) GetPayment(req GetPaymentRequest) (*Payment, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_get_payment(
			_pointer, FfiConverterTypeGetPaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalTypePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListFiatCurrencies() ([]FiatCurrency, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_fiat_currencies(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []FiatCurrency
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeFiatCurrencyINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListPayments(req ListPaymentsRequest) ([]Payment, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_payments(
			_pointer, FfiConverterTypeListPaymentsRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ListRefundables() ([]RefundableSwap, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_list_refundables(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []RefundableSwap
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeRefundableSwapINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlAuthError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_auth(
			_pointer, FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Lower(reqData), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlCallbackStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlCallbackStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlPay(req LnUrlPayRequest) (LnUrlPayResult, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_pay(
			_pointer, FfiConverterTypeLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlPayResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlPayResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) LnurlWithdraw(req LnUrlWithdrawRequest) (LnUrlWithdrawResult, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlWithdrawError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_lnurl_withdraw(
			_pointer, FfiConverterTypeLnUrlWithdrawRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlWithdrawResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlWithdrawResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Parse(input string) (InputType, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_parse(
			_pointer, FfiConverterStringINSTANCE.Lower(input), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue InputType
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeInputTypeINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PayOnchain(req PayOnchainRequest) (SendPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_pay_onchain(
			_pointer, FfiConverterTypePayOnchainRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareBuyBitcoin(req PrepareBuyBitcoinRequest) (PrepareBuyBitcoinResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_buy_bitcoin(
			_pointer, FfiConverterTypePrepareBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareBuyBitcoinResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareBuyBitcoinResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareLnurlPay(req PrepareLnUrlPayRequest) (PrepareLnUrlPayResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_lnurl_pay(
			_pointer, FfiConverterTypePrepareLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareLnUrlPayResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareLnUrlPayResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PreparePayOnchain(req PreparePayOnchainRequest) (PreparePayOnchainResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_pay_onchain(
			_pointer, FfiConverterTypePreparePayOnchainRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PreparePayOnchainResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePreparePayOnchainResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareReceivePayment(req PrepareReceiveRequest) (PrepareReceiveResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_receive_payment(
			_pointer, FfiConverterTypePrepareReceiveRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareReceiveResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareReceiveResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_refund(
			_pointer, FfiConverterTypePrepareRefundRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareSendPayment(req PrepareSendRequest) (PrepareSendResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_prepare_send_payment(
			_pointer, FfiConverterTypePrepareSendRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareSendResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareSendResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_receive_payment(
			_pointer, FfiConverterTypeReceivePaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReceivePaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeReceivePaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) RecommendedFees() (RecommendedFees, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_recommended_fees(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RecommendedFees
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeRecommendedFeesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Refund(req RefundRequest) (RefundResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_refund(
			_pointer, FfiConverterTypeRefundRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) RegisterWebhook(webhookUrl string) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_register_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) RemoveEventListener(id string) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_remove_event_listener(
			_pointer, FfiConverterStringINSTANCE.Lower(id), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) RescanOnchainSwaps() error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_rescan_onchain_swaps(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) Restore(req RestoreRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_restore(
			_pointer, FfiConverterTypeRestoreRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) SendPayment(req SendPaymentRequest) (SendPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_send_payment(
			_pointer, FfiConverterTypeSendPaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) SignMessage(req SignMessageRequest) (SignMessageResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_sign_message(
			_pointer, FfiConverterTypeSignMessageRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SignMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSignMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Sync() error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_method_bindingliquidsdk_sync(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) UnregisterWebhook() error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
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
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_breez_sdk_liquid_bindings_fn_free_bindingliquidsdk(pointer, status)
			}),
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
	FfiDestroyerTypeFetchPaymentProposedFeesResponse{}.Destroy(r.Response)
}

type FfiConverterTypeAcceptPaymentProposedFeesRequest struct{}

var FfiConverterTypeAcceptPaymentProposedFeesRequestINSTANCE = FfiConverterTypeAcceptPaymentProposedFeesRequest{}

func (c FfiConverterTypeAcceptPaymentProposedFeesRequest) Lift(rb RustBufferI) AcceptPaymentProposedFeesRequest {
	return LiftFromRustBuffer[AcceptPaymentProposedFeesRequest](c, rb)
}

func (c FfiConverterTypeAcceptPaymentProposedFeesRequest) Read(reader io.Reader) AcceptPaymentProposedFeesRequest {
	return AcceptPaymentProposedFeesRequest{
		FfiConverterTypeFetchPaymentProposedFeesResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAcceptPaymentProposedFeesRequest) Lower(value AcceptPaymentProposedFeesRequest) RustBuffer {
	return LowerIntoRustBuffer[AcceptPaymentProposedFeesRequest](c, value)
}

func (c FfiConverterTypeAcceptPaymentProposedFeesRequest) Write(writer io.Writer, value AcceptPaymentProposedFeesRequest) {
	FfiConverterTypeFetchPaymentProposedFeesResponseINSTANCE.Write(writer, value.Response)
}

type FfiDestroyerTypeAcceptPaymentProposedFeesRequest struct{}

func (_ FfiDestroyerTypeAcceptPaymentProposedFeesRequest) Destroy(value AcceptPaymentProposedFeesRequest) {
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

type FfiConverterTypeAesSuccessActionData struct{}

var FfiConverterTypeAesSuccessActionDataINSTANCE = FfiConverterTypeAesSuccessActionData{}

func (c FfiConverterTypeAesSuccessActionData) Lift(rb RustBufferI) AesSuccessActionData {
	return LiftFromRustBuffer[AesSuccessActionData](c, rb)
}

func (c FfiConverterTypeAesSuccessActionData) Read(reader io.Reader) AesSuccessActionData {
	return AesSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAesSuccessActionData) Lower(value AesSuccessActionData) RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionData](c, value)
}

func (c FfiConverterTypeAesSuccessActionData) Write(writer io.Writer, value AesSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Ciphertext)
	FfiConverterStringINSTANCE.Write(writer, value.Iv)
}

type FfiDestroyerTypeAesSuccessActionData struct{}

func (_ FfiDestroyerTypeAesSuccessActionData) Destroy(value AesSuccessActionData) {
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

type FfiConverterTypeAesSuccessActionDataDecrypted struct{}

var FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE = FfiConverterTypeAesSuccessActionDataDecrypted{}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Lift(rb RustBufferI) AesSuccessActionDataDecrypted {
	return LiftFromRustBuffer[AesSuccessActionDataDecrypted](c, rb)
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Read(reader io.Reader) AesSuccessActionDataDecrypted {
	return AesSuccessActionDataDecrypted{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Lower(value AesSuccessActionDataDecrypted) RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataDecrypted](c, value)
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Write(writer io.Writer, value AesSuccessActionDataDecrypted) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Plaintext)
}

type FfiDestroyerTypeAesSuccessActionDataDecrypted struct{}

func (_ FfiDestroyerTypeAesSuccessActionDataDecrypted) Destroy(value AesSuccessActionDataDecrypted) {
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

type FfiConverterTypeAssetBalance struct{}

var FfiConverterTypeAssetBalanceINSTANCE = FfiConverterTypeAssetBalance{}

func (c FfiConverterTypeAssetBalance) Lift(rb RustBufferI) AssetBalance {
	return LiftFromRustBuffer[AssetBalance](c, rb)
}

func (c FfiConverterTypeAssetBalance) Read(reader io.Reader) AssetBalance {
	return AssetBalance{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAssetBalance) Lower(value AssetBalance) RustBuffer {
	return LowerIntoRustBuffer[AssetBalance](c, value)
}

func (c FfiConverterTypeAssetBalance) Write(writer io.Writer, value AssetBalance) {
	FfiConverterStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterUint64INSTANCE.Write(writer, value.BalanceSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Name)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.Balance)
}

type FfiDestroyerTypeAssetBalance struct{}

func (_ FfiDestroyerTypeAssetBalance) Destroy(value AssetBalance) {
	value.Destroy()
}

type AssetInfo struct {
	Name   string
	Ticker string
	Amount float64
}

func (r *AssetInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerString{}.Destroy(r.Ticker)
	FfiDestroyerFloat64{}.Destroy(r.Amount)
}

type FfiConverterTypeAssetInfo struct{}

var FfiConverterTypeAssetInfoINSTANCE = FfiConverterTypeAssetInfo{}

func (c FfiConverterTypeAssetInfo) Lift(rb RustBufferI) AssetInfo {
	return LiftFromRustBuffer[AssetInfo](c, rb)
}

func (c FfiConverterTypeAssetInfo) Read(reader io.Reader) AssetInfo {
	return AssetInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAssetInfo) Lower(value AssetInfo) RustBuffer {
	return LowerIntoRustBuffer[AssetInfo](c, value)
}

func (c FfiConverterTypeAssetInfo) Write(writer io.Writer, value AssetInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterFloat64INSTANCE.Write(writer, value.Amount)
}

type FfiDestroyerTypeAssetInfo struct{}

func (_ FfiDestroyerTypeAssetInfo) Destroy(value AssetInfo) {
	value.Destroy()
}

type AssetMetadata struct {
	AssetId   string
	Name      string
	Ticker    string
	Precision uint8
}

func (r *AssetMetadata) Destroy() {
	FfiDestroyerString{}.Destroy(r.AssetId)
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerString{}.Destroy(r.Ticker)
	FfiDestroyerUint8{}.Destroy(r.Precision)
}

type FfiConverterTypeAssetMetadata struct{}

var FfiConverterTypeAssetMetadataINSTANCE = FfiConverterTypeAssetMetadata{}

func (c FfiConverterTypeAssetMetadata) Lift(rb RustBufferI) AssetMetadata {
	return LiftFromRustBuffer[AssetMetadata](c, rb)
}

func (c FfiConverterTypeAssetMetadata) Read(reader io.Reader) AssetMetadata {
	return AssetMetadata{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAssetMetadata) Lower(value AssetMetadata) RustBuffer {
	return LowerIntoRustBuffer[AssetMetadata](c, value)
}

func (c FfiConverterTypeAssetMetadata) Write(writer io.Writer, value AssetMetadata) {
	FfiConverterStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterStringINSTANCE.Write(writer, value.Ticker)
	FfiConverterUint8INSTANCE.Write(writer, value.Precision)
}

type FfiDestroyerTypeAssetMetadata struct{}

func (_ FfiDestroyerTypeAssetMetadata) Destroy(value AssetMetadata) {
	value.Destroy()
}

type BackupRequest struct {
	BackupPath *string
}

func (r *BackupRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.BackupPath)
}

type FfiConverterTypeBackupRequest struct{}

var FfiConverterTypeBackupRequestINSTANCE = FfiConverterTypeBackupRequest{}

func (c FfiConverterTypeBackupRequest) Lift(rb RustBufferI) BackupRequest {
	return LiftFromRustBuffer[BackupRequest](c, rb)
}

func (c FfiConverterTypeBackupRequest) Read(reader io.Reader) BackupRequest {
	return BackupRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBackupRequest) Lower(value BackupRequest) RustBuffer {
	return LowerIntoRustBuffer[BackupRequest](c, value)
}

func (c FfiConverterTypeBackupRequest) Write(writer io.Writer, value BackupRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BackupPath)
}

type FfiDestroyerTypeBackupRequest struct{}

func (_ FfiDestroyerTypeBackupRequest) Destroy(value BackupRequest) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterTypeBitcoinAddressData struct{}

var FfiConverterTypeBitcoinAddressDataINSTANCE = FfiConverterTypeBitcoinAddressData{}

func (c FfiConverterTypeBitcoinAddressData) Lift(rb RustBufferI) BitcoinAddressData {
	return LiftFromRustBuffer[BitcoinAddressData](c, rb)
}

func (c FfiConverterTypeBitcoinAddressData) Read(reader io.Reader) BitcoinAddressData {
	return BitcoinAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBitcoinAddressData) Lower(value BitcoinAddressData) RustBuffer {
	return LowerIntoRustBuffer[BitcoinAddressData](c, value)
}

func (c FfiConverterTypeBitcoinAddressData) Write(writer io.Writer, value BitcoinAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeBitcoinAddressData struct{}

func (_ FfiDestroyerTypeBitcoinAddressData) Destroy(value BitcoinAddressData) {
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

type FfiConverterTypeBlockchainInfo struct{}

var FfiConverterTypeBlockchainInfoINSTANCE = FfiConverterTypeBlockchainInfo{}

func (c FfiConverterTypeBlockchainInfo) Lift(rb RustBufferI) BlockchainInfo {
	return LiftFromRustBuffer[BlockchainInfo](c, rb)
}

func (c FfiConverterTypeBlockchainInfo) Read(reader io.Reader) BlockchainInfo {
	return BlockchainInfo{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBlockchainInfo) Lower(value BlockchainInfo) RustBuffer {
	return LowerIntoRustBuffer[BlockchainInfo](c, value)
}

func (c FfiConverterTypeBlockchainInfo) Write(writer io.Writer, value BlockchainInfo) {
	FfiConverterUint32INSTANCE.Write(writer, value.LiquidTip)
	FfiConverterUint32INSTANCE.Write(writer, value.BitcoinTip)
}

type FfiDestroyerTypeBlockchainInfo struct{}

func (_ FfiDestroyerTypeBlockchainInfo) Destroy(value BlockchainInfo) {
	value.Destroy()
}

type BuyBitcoinRequest struct {
	PrepareResponse PrepareBuyBitcoinResponse
	RedirectUrl     *string
}

func (r *BuyBitcoinRequest) Destroy() {
	FfiDestroyerTypePrepareBuyBitcoinResponse{}.Destroy(r.PrepareResponse)
	FfiDestroyerOptionalString{}.Destroy(r.RedirectUrl)
}

type FfiConverterTypeBuyBitcoinRequest struct{}

var FfiConverterTypeBuyBitcoinRequestINSTANCE = FfiConverterTypeBuyBitcoinRequest{}

func (c FfiConverterTypeBuyBitcoinRequest) Lift(rb RustBufferI) BuyBitcoinRequest {
	return LiftFromRustBuffer[BuyBitcoinRequest](c, rb)
}

func (c FfiConverterTypeBuyBitcoinRequest) Read(reader io.Reader) BuyBitcoinRequest {
	return BuyBitcoinRequest{
		FfiConverterTypePrepareBuyBitcoinResponseINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBuyBitcoinRequest) Lower(value BuyBitcoinRequest) RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinRequest](c, value)
}

func (c FfiConverterTypeBuyBitcoinRequest) Write(writer io.Writer, value BuyBitcoinRequest) {
	FfiConverterTypePrepareBuyBitcoinResponseINSTANCE.Write(writer, value.PrepareResponse)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.RedirectUrl)
}

type FfiDestroyerTypeBuyBitcoinRequest struct{}

func (_ FfiDestroyerTypeBuyBitcoinRequest) Destroy(value BuyBitcoinRequest) {
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

type FfiConverterTypeCheckMessageRequest struct{}

var FfiConverterTypeCheckMessageRequestINSTANCE = FfiConverterTypeCheckMessageRequest{}

func (c FfiConverterTypeCheckMessageRequest) Lift(rb RustBufferI) CheckMessageRequest {
	return LiftFromRustBuffer[CheckMessageRequest](c, rb)
}

func (c FfiConverterTypeCheckMessageRequest) Read(reader io.Reader) CheckMessageRequest {
	return CheckMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCheckMessageRequest) Lower(value CheckMessageRequest) RustBuffer {
	return LowerIntoRustBuffer[CheckMessageRequest](c, value)
}

func (c FfiConverterTypeCheckMessageRequest) Write(writer io.Writer, value CheckMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerTypeCheckMessageRequest struct{}

func (_ FfiDestroyerTypeCheckMessageRequest) Destroy(value CheckMessageRequest) {
	value.Destroy()
}

type CheckMessageResponse struct {
	IsValid bool
}

func (r *CheckMessageResponse) Destroy() {
	FfiDestroyerBool{}.Destroy(r.IsValid)
}

type FfiConverterTypeCheckMessageResponse struct{}

var FfiConverterTypeCheckMessageResponseINSTANCE = FfiConverterTypeCheckMessageResponse{}

func (c FfiConverterTypeCheckMessageResponse) Lift(rb RustBufferI) CheckMessageResponse {
	return LiftFromRustBuffer[CheckMessageResponse](c, rb)
}

func (c FfiConverterTypeCheckMessageResponse) Read(reader io.Reader) CheckMessageResponse {
	return CheckMessageResponse{
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCheckMessageResponse) Lower(value CheckMessageResponse) RustBuffer {
	return LowerIntoRustBuffer[CheckMessageResponse](c, value)
}

func (c FfiConverterTypeCheckMessageResponse) Write(writer io.Writer, value CheckMessageResponse) {
	FfiConverterBoolINSTANCE.Write(writer, value.IsValid)
}

type FfiDestroyerTypeCheckMessageResponse struct{}

func (_ FfiDestroyerTypeCheckMessageResponse) Destroy(value CheckMessageResponse) {
	value.Destroy()
}

type Config struct {
	LiquidElectrumUrl               string
	BitcoinElectrumUrl              string
	MempoolspaceUrl                 string
	WorkingDir                      string
	Network                         LiquidNetwork
	PaymentTimeoutSec               uint64
	ZeroConfMinFeeRateMsat          uint32
	SyncServiceUrl                  *string
	BreezApiKey                     *string
	CacheDir                        *string
	ZeroConfMaxAmountSat            *uint64
	UseDefaultExternalInputParsers  bool
	ExternalInputParsers            *[]ExternalInputParser
	OnchainFeeRateLeewaySatPerVbyte *uint32
	AssetMetadata                   *[]AssetMetadata
}

func (r *Config) Destroy() {
	FfiDestroyerString{}.Destroy(r.LiquidElectrumUrl)
	FfiDestroyerString{}.Destroy(r.BitcoinElectrumUrl)
	FfiDestroyerString{}.Destroy(r.MempoolspaceUrl)
	FfiDestroyerString{}.Destroy(r.WorkingDir)
	FfiDestroyerTypeLiquidNetwork{}.Destroy(r.Network)
	FfiDestroyerUint64{}.Destroy(r.PaymentTimeoutSec)
	FfiDestroyerUint32{}.Destroy(r.ZeroConfMinFeeRateMsat)
	FfiDestroyerOptionalString{}.Destroy(r.SyncServiceUrl)
	FfiDestroyerOptionalString{}.Destroy(r.BreezApiKey)
	FfiDestroyerOptionalString{}.Destroy(r.CacheDir)
	FfiDestroyerOptionalUint64{}.Destroy(r.ZeroConfMaxAmountSat)
	FfiDestroyerBool{}.Destroy(r.UseDefaultExternalInputParsers)
	FfiDestroyerOptionalSequenceTypeExternalInputParser{}.Destroy(r.ExternalInputParsers)
	FfiDestroyerOptionalUint32{}.Destroy(r.OnchainFeeRateLeewaySatPerVbyte)
	FfiDestroyerOptionalSequenceTypeAssetMetadata{}.Destroy(r.AssetMetadata)
}

type FfiConverterTypeConfig struct{}

var FfiConverterTypeConfigINSTANCE = FfiConverterTypeConfig{}

func (c FfiConverterTypeConfig) Lift(rb RustBufferI) Config {
	return LiftFromRustBuffer[Config](c, rb)
}

func (c FfiConverterTypeConfig) Read(reader io.Reader) Config {
	return Config{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeLiquidNetworkINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalSequenceTypeExternalInputParserINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalSequenceTypeAssetMetadataINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConfig) Lower(value Config) RustBuffer {
	return LowerIntoRustBuffer[Config](c, value)
}

func (c FfiConverterTypeConfig) Write(writer io.Writer, value Config) {
	FfiConverterStringINSTANCE.Write(writer, value.LiquidElectrumUrl)
	FfiConverterStringINSTANCE.Write(writer, value.BitcoinElectrumUrl)
	FfiConverterStringINSTANCE.Write(writer, value.MempoolspaceUrl)
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
	FfiConverterTypeLiquidNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterUint64INSTANCE.Write(writer, value.PaymentTimeoutSec)
	FfiConverterUint32INSTANCE.Write(writer, value.ZeroConfMinFeeRateMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.SyncServiceUrl)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BreezApiKey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.CacheDir)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.ZeroConfMaxAmountSat)
	FfiConverterBoolINSTANCE.Write(writer, value.UseDefaultExternalInputParsers)
	FfiConverterOptionalSequenceTypeExternalInputParserINSTANCE.Write(writer, value.ExternalInputParsers)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.OnchainFeeRateLeewaySatPerVbyte)
	FfiConverterOptionalSequenceTypeAssetMetadataINSTANCE.Write(writer, value.AssetMetadata)
}

type FfiDestroyerTypeConfig struct{}

func (_ FfiDestroyerTypeConfig) Destroy(value Config) {
	value.Destroy()
}

type ConnectRequest struct {
	Config   Config
	Mnemonic string
}

func (r *ConnectRequest) Destroy() {
	FfiDestroyerTypeConfig{}.Destroy(r.Config)
	FfiDestroyerString{}.Destroy(r.Mnemonic)
}

type FfiConverterTypeConnectRequest struct{}

var FfiConverterTypeConnectRequestINSTANCE = FfiConverterTypeConnectRequest{}

func (c FfiConverterTypeConnectRequest) Lift(rb RustBufferI) ConnectRequest {
	return LiftFromRustBuffer[ConnectRequest](c, rb)
}

func (c FfiConverterTypeConnectRequest) Read(reader io.Reader) ConnectRequest {
	return ConnectRequest{
		FfiConverterTypeConfigINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectRequest) Lower(value ConnectRequest) RustBuffer {
	return LowerIntoRustBuffer[ConnectRequest](c, value)
}

func (c FfiConverterTypeConnectRequest) Write(writer io.Writer, value ConnectRequest) {
	FfiConverterTypeConfigINSTANCE.Write(writer, value.Config)
	FfiConverterStringINSTANCE.Write(writer, value.Mnemonic)
}

type FfiDestroyerTypeConnectRequest struct{}

func (_ FfiDestroyerTypeConnectRequest) Destroy(value ConnectRequest) {
	value.Destroy()
}

type ConnectWithSignerRequest struct {
	Config Config
}

func (r *ConnectWithSignerRequest) Destroy() {
	FfiDestroyerTypeConfig{}.Destroy(r.Config)
}

type FfiConverterTypeConnectWithSignerRequest struct{}

var FfiConverterTypeConnectWithSignerRequestINSTANCE = FfiConverterTypeConnectWithSignerRequest{}

func (c FfiConverterTypeConnectWithSignerRequest) Lift(rb RustBufferI) ConnectWithSignerRequest {
	return LiftFromRustBuffer[ConnectWithSignerRequest](c, rb)
}

func (c FfiConverterTypeConnectWithSignerRequest) Read(reader io.Reader) ConnectWithSignerRequest {
	return ConnectWithSignerRequest{
		FfiConverterTypeConfigINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectWithSignerRequest) Lower(value ConnectWithSignerRequest) RustBuffer {
	return LowerIntoRustBuffer[ConnectWithSignerRequest](c, value)
}

func (c FfiConverterTypeConnectWithSignerRequest) Write(writer io.Writer, value ConnectWithSignerRequest) {
	FfiConverterTypeConfigINSTANCE.Write(writer, value.Config)
}

type FfiDestroyerTypeConnectWithSignerRequest struct{}

func (_ FfiDestroyerTypeConnectWithSignerRequest) Destroy(value ConnectWithSignerRequest) {
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
	FfiDestroyerOptionalTypeSymbol{}.Destroy(r.Symbol)
	FfiDestroyerOptionalTypeSymbol{}.Destroy(r.UniqSymbol)
	FfiDestroyerSequenceTypeLocalizedName{}.Destroy(r.LocalizedName)
	FfiDestroyerSequenceTypeLocaleOverrides{}.Destroy(r.LocaleOverrides)
}

type FfiConverterTypeCurrencyInfo struct{}

var FfiConverterTypeCurrencyInfoINSTANCE = FfiConverterTypeCurrencyInfo{}

func (c FfiConverterTypeCurrencyInfo) Lift(rb RustBufferI) CurrencyInfo {
	return LiftFromRustBuffer[CurrencyInfo](c, rb)
}

func (c FfiConverterTypeCurrencyInfo) Read(reader io.Reader) CurrencyInfo {
	return CurrencyInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalTypeSymbolINSTANCE.Read(reader),
		FfiConverterOptionalTypeSymbolINSTANCE.Read(reader),
		FfiConverterSequenceTypeLocalizedNameINSTANCE.Read(reader),
		FfiConverterSequenceTypeLocaleOverridesINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCurrencyInfo) Lower(value CurrencyInfo) RustBuffer {
	return LowerIntoRustBuffer[CurrencyInfo](c, value)
}

func (c FfiConverterTypeCurrencyInfo) Write(writer io.Writer, value CurrencyInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterUint32INSTANCE.Write(writer, value.FractionSize)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterOptionalTypeSymbolINSTANCE.Write(writer, value.Symbol)
	FfiConverterOptionalTypeSymbolINSTANCE.Write(writer, value.UniqSymbol)
	FfiConverterSequenceTypeLocalizedNameINSTANCE.Write(writer, value.LocalizedName)
	FfiConverterSequenceTypeLocaleOverridesINSTANCE.Write(writer, value.LocaleOverrides)
}

type FfiDestroyerTypeCurrencyInfo struct{}

func (_ FfiDestroyerTypeCurrencyInfo) Destroy(value CurrencyInfo) {
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

type FfiConverterTypeExternalInputParser struct{}

var FfiConverterTypeExternalInputParserINSTANCE = FfiConverterTypeExternalInputParser{}

func (c FfiConverterTypeExternalInputParser) Lift(rb RustBufferI) ExternalInputParser {
	return LiftFromRustBuffer[ExternalInputParser](c, rb)
}

func (c FfiConverterTypeExternalInputParser) Read(reader io.Reader) ExternalInputParser {
	return ExternalInputParser{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeExternalInputParser) Lower(value ExternalInputParser) RustBuffer {
	return LowerIntoRustBuffer[ExternalInputParser](c, value)
}

func (c FfiConverterTypeExternalInputParser) Write(writer io.Writer, value ExternalInputParser) {
	FfiConverterStringINSTANCE.Write(writer, value.ProviderId)
	FfiConverterStringINSTANCE.Write(writer, value.InputRegex)
	FfiConverterStringINSTANCE.Write(writer, value.ParserUrl)
}

type FfiDestroyerTypeExternalInputParser struct{}

func (_ FfiDestroyerTypeExternalInputParser) Destroy(value ExternalInputParser) {
	value.Destroy()
}

type FetchPaymentProposedFeesRequest struct {
	SwapId string
}

func (r *FetchPaymentProposedFeesRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapId)
}

type FfiConverterTypeFetchPaymentProposedFeesRequest struct{}

var FfiConverterTypeFetchPaymentProposedFeesRequestINSTANCE = FfiConverterTypeFetchPaymentProposedFeesRequest{}

func (c FfiConverterTypeFetchPaymentProposedFeesRequest) Lift(rb RustBufferI) FetchPaymentProposedFeesRequest {
	return LiftFromRustBuffer[FetchPaymentProposedFeesRequest](c, rb)
}

func (c FfiConverterTypeFetchPaymentProposedFeesRequest) Read(reader io.Reader) FetchPaymentProposedFeesRequest {
	return FetchPaymentProposedFeesRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFetchPaymentProposedFeesRequest) Lower(value FetchPaymentProposedFeesRequest) RustBuffer {
	return LowerIntoRustBuffer[FetchPaymentProposedFeesRequest](c, value)
}

func (c FfiConverterTypeFetchPaymentProposedFeesRequest) Write(writer io.Writer, value FetchPaymentProposedFeesRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapId)
}

type FfiDestroyerTypeFetchPaymentProposedFeesRequest struct{}

func (_ FfiDestroyerTypeFetchPaymentProposedFeesRequest) Destroy(value FetchPaymentProposedFeesRequest) {
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

type FfiConverterTypeFetchPaymentProposedFeesResponse struct{}

var FfiConverterTypeFetchPaymentProposedFeesResponseINSTANCE = FfiConverterTypeFetchPaymentProposedFeesResponse{}

func (c FfiConverterTypeFetchPaymentProposedFeesResponse) Lift(rb RustBufferI) FetchPaymentProposedFeesResponse {
	return LiftFromRustBuffer[FetchPaymentProposedFeesResponse](c, rb)
}

func (c FfiConverterTypeFetchPaymentProposedFeesResponse) Read(reader io.Reader) FetchPaymentProposedFeesResponse {
	return FetchPaymentProposedFeesResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFetchPaymentProposedFeesResponse) Lower(value FetchPaymentProposedFeesResponse) RustBuffer {
	return LowerIntoRustBuffer[FetchPaymentProposedFeesResponse](c, value)
}

func (c FfiConverterTypeFetchPaymentProposedFeesResponse) Write(writer io.Writer, value FetchPaymentProposedFeesResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapId)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PayerAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.ReceiverAmountSat)
}

type FfiDestroyerTypeFetchPaymentProposedFeesResponse struct{}

func (_ FfiDestroyerTypeFetchPaymentProposedFeesResponse) Destroy(value FetchPaymentProposedFeesResponse) {
	value.Destroy()
}

type FiatCurrency struct {
	Id   string
	Info CurrencyInfo
}

func (r *FiatCurrency) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerTypeCurrencyInfo{}.Destroy(r.Info)
}

type FfiConverterTypeFiatCurrency struct{}

var FfiConverterTypeFiatCurrencyINSTANCE = FfiConverterTypeFiatCurrency{}

func (c FfiConverterTypeFiatCurrency) Lift(rb RustBufferI) FiatCurrency {
	return LiftFromRustBuffer[FiatCurrency](c, rb)
}

func (c FfiConverterTypeFiatCurrency) Read(reader io.Reader) FiatCurrency {
	return FiatCurrency{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeCurrencyInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFiatCurrency) Lower(value FiatCurrency) RustBuffer {
	return LowerIntoRustBuffer[FiatCurrency](c, value)
}

func (c FfiConverterTypeFiatCurrency) Write(writer io.Writer, value FiatCurrency) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterTypeCurrencyInfoINSTANCE.Write(writer, value.Info)
}

type FfiDestroyerTypeFiatCurrency struct{}

func (_ FfiDestroyerTypeFiatCurrency) Destroy(value FiatCurrency) {
	value.Destroy()
}

type GetInfoResponse struct {
	WalletInfo     WalletInfo
	BlockchainInfo BlockchainInfo
}

func (r *GetInfoResponse) Destroy() {
	FfiDestroyerTypeWalletInfo{}.Destroy(r.WalletInfo)
	FfiDestroyerTypeBlockchainInfo{}.Destroy(r.BlockchainInfo)
}

type FfiConverterTypeGetInfoResponse struct{}

var FfiConverterTypeGetInfoResponseINSTANCE = FfiConverterTypeGetInfoResponse{}

func (c FfiConverterTypeGetInfoResponse) Lift(rb RustBufferI) GetInfoResponse {
	return LiftFromRustBuffer[GetInfoResponse](c, rb)
}

func (c FfiConverterTypeGetInfoResponse) Read(reader io.Reader) GetInfoResponse {
	return GetInfoResponse{
		FfiConverterTypeWalletInfoINSTANCE.Read(reader),
		FfiConverterTypeBlockchainInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGetInfoResponse) Lower(value GetInfoResponse) RustBuffer {
	return LowerIntoRustBuffer[GetInfoResponse](c, value)
}

func (c FfiConverterTypeGetInfoResponse) Write(writer io.Writer, value GetInfoResponse) {
	FfiConverterTypeWalletInfoINSTANCE.Write(writer, value.WalletInfo)
	FfiConverterTypeBlockchainInfoINSTANCE.Write(writer, value.BlockchainInfo)
}

type FfiDestroyerTypeGetInfoResponse struct{}

func (_ FfiDestroyerTypeGetInfoResponse) Destroy(value GetInfoResponse) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerString{}.Destroy(r.PayeePubkey)
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.DescriptionHash)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
	FfiDestroyerSequenceTypeRouteHint{}.Destroy(r.RoutingHints)
	FfiDestroyerSequenceUint8{}.Destroy(r.PaymentSecret)
	FfiDestroyerUint64{}.Destroy(r.MinFinalCltvExpiryDelta)
}

type FfiConverterTypeLNInvoice struct{}

var FfiConverterTypeLNInvoiceINSTANCE = FfiConverterTypeLNInvoice{}

func (c FfiConverterTypeLNInvoice) Lift(rb RustBufferI) LnInvoice {
	return LiftFromRustBuffer[LnInvoice](c, rb)
}

func (c FfiConverterTypeLNInvoice) Read(reader io.Reader) LnInvoice {
	return LnInvoice{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceTypeRouteHintINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLNInvoice) Lower(value LnInvoice) RustBuffer {
	return LowerIntoRustBuffer[LnInvoice](c, value)
}

func (c FfiConverterTypeLNInvoice) Write(writer io.Writer, value LnInvoice) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterStringINSTANCE.Write(writer, value.PayeePubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DescriptionHash)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
	FfiConverterSequenceTypeRouteHintINSTANCE.Write(writer, value.RoutingHints)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PaymentSecret)
	FfiConverterUint64INSTANCE.Write(writer, value.MinFinalCltvExpiryDelta)
}

type FfiDestroyerTypeLnInvoice struct{}

func (_ FfiDestroyerTypeLnInvoice) Destroy(value LnInvoice) {
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
	FfiDestroyerSequenceTypeLnOfferBlindedPath{}.Destroy(r.Paths)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.SigningPubkey)
	FfiDestroyerOptionalTypeAmount{}.Destroy(r.MinAmount)
	FfiDestroyerOptionalUint64{}.Destroy(r.AbsoluteExpiry)
	FfiDestroyerOptionalString{}.Destroy(r.Issuer)
}

type FfiConverterTypeLNOffer struct{}

var FfiConverterTypeLNOfferINSTANCE = FfiConverterTypeLNOffer{}

func (c FfiConverterTypeLNOffer) Lift(rb RustBufferI) LnOffer {
	return LiftFromRustBuffer[LnOffer](c, rb)
}

func (c FfiConverterTypeLNOffer) Read(reader io.Reader) LnOffer {
	return LnOffer{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceTypeLnOfferBlindedPathINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLNOffer) Lower(value LnOffer) RustBuffer {
	return LowerIntoRustBuffer[LnOffer](c, value)
}

func (c FfiConverterTypeLNOffer) Write(writer io.Writer, value LnOffer) {
	FfiConverterStringINSTANCE.Write(writer, value.Offer)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.Chains)
	FfiConverterSequenceTypeLnOfferBlindedPathINSTANCE.Write(writer, value.Paths)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.SigningPubkey)
	FfiConverterOptionalTypeAmountINSTANCE.Write(writer, value.MinAmount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AbsoluteExpiry)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Issuer)
}

type FfiDestroyerTypeLnOffer struct{}

func (_ FfiDestroyerTypeLnOffer) Destroy(value LnOffer) {
	value.Destroy()
}

type LightningPaymentLimitsResponse struct {
	Send    Limits
	Receive Limits
}

func (r *LightningPaymentLimitsResponse) Destroy() {
	FfiDestroyerTypeLimits{}.Destroy(r.Send)
	FfiDestroyerTypeLimits{}.Destroy(r.Receive)
}

type FfiConverterTypeLightningPaymentLimitsResponse struct{}

var FfiConverterTypeLightningPaymentLimitsResponseINSTANCE = FfiConverterTypeLightningPaymentLimitsResponse{}

func (c FfiConverterTypeLightningPaymentLimitsResponse) Lift(rb RustBufferI) LightningPaymentLimitsResponse {
	return LiftFromRustBuffer[LightningPaymentLimitsResponse](c, rb)
}

func (c FfiConverterTypeLightningPaymentLimitsResponse) Read(reader io.Reader) LightningPaymentLimitsResponse {
	return LightningPaymentLimitsResponse{
		FfiConverterTypeLimitsINSTANCE.Read(reader),
		FfiConverterTypeLimitsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLightningPaymentLimitsResponse) Lower(value LightningPaymentLimitsResponse) RustBuffer {
	return LowerIntoRustBuffer[LightningPaymentLimitsResponse](c, value)
}

func (c FfiConverterTypeLightningPaymentLimitsResponse) Write(writer io.Writer, value LightningPaymentLimitsResponse) {
	FfiConverterTypeLimitsINSTANCE.Write(writer, value.Send)
	FfiConverterTypeLimitsINSTANCE.Write(writer, value.Receive)
}

type FfiDestroyerTypeLightningPaymentLimitsResponse struct{}

func (_ FfiDestroyerTypeLightningPaymentLimitsResponse) Destroy(value LightningPaymentLimitsResponse) {
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

type FfiConverterTypeLimits struct{}

var FfiConverterTypeLimitsINSTANCE = FfiConverterTypeLimits{}

func (c FfiConverterTypeLimits) Lift(rb RustBufferI) Limits {
	return LiftFromRustBuffer[Limits](c, rb)
}

func (c FfiConverterTypeLimits) Read(reader io.Reader) Limits {
	return Limits{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLimits) Lower(value Limits) RustBuffer {
	return LowerIntoRustBuffer[Limits](c, value)
}

func (c FfiConverterTypeLimits) Write(writer io.Writer, value Limits) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxZeroConfSat)
}

type FfiDestroyerTypeLimits struct{}

func (_ FfiDestroyerTypeLimits) Destroy(value Limits) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalString{}.Destroy(r.AssetId)
	FfiDestroyerOptionalFloat64{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterTypeLiquidAddressData struct{}

var FfiConverterTypeLiquidAddressDataINSTANCE = FfiConverterTypeLiquidAddressData{}

func (c FfiConverterTypeLiquidAddressData) Lift(rb RustBufferI) LiquidAddressData {
	return LiftFromRustBuffer[LiquidAddressData](c, rb)
}

func (c FfiConverterTypeLiquidAddressData) Read(reader io.Reader) LiquidAddressData {
	return LiquidAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLiquidAddressData) Lower(value LiquidAddressData) RustBuffer {
	return LowerIntoRustBuffer[LiquidAddressData](c, value)
}

func (c FfiConverterTypeLiquidAddressData) Write(writer io.Writer, value LiquidAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.AssetId)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeLiquidAddressData struct{}

func (_ FfiDestroyerTypeLiquidAddressData) Destroy(value LiquidAddressData) {
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
	FfiDestroyerOptionalSequenceTypePaymentType{}.Destroy(r.Filters)
	FfiDestroyerOptionalSequenceTypePaymentState{}.Destroy(r.States)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
	FfiDestroyerOptionalTypeListPaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalBool{}.Destroy(r.SortAscending)
}

type FfiConverterTypeListPaymentsRequest struct{}

var FfiConverterTypeListPaymentsRequestINSTANCE = FfiConverterTypeListPaymentsRequest{}

func (c FfiConverterTypeListPaymentsRequest) Lift(rb RustBufferI) ListPaymentsRequest {
	return LiftFromRustBuffer[ListPaymentsRequest](c, rb)
}

func (c FfiConverterTypeListPaymentsRequest) Read(reader io.Reader) ListPaymentsRequest {
	return ListPaymentsRequest{
		FfiConverterOptionalSequenceTypePaymentTypeINSTANCE.Read(reader),
		FfiConverterOptionalSequenceTypePaymentStateINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalTypeListPaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListPaymentsRequest) Lower(value ListPaymentsRequest) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsRequest](c, value)
}

func (c FfiConverterTypeListPaymentsRequest) Write(writer io.Writer, value ListPaymentsRequest) {
	FfiConverterOptionalSequenceTypePaymentTypeINSTANCE.Write(writer, value.Filters)
	FfiConverterOptionalSequenceTypePaymentStateINSTANCE.Write(writer, value.States)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
	FfiConverterOptionalTypeListPaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.SortAscending)
}

type FfiDestroyerTypeListPaymentsRequest struct{}

func (_ FfiDestroyerTypeListPaymentsRequest) Destroy(value ListPaymentsRequest) {
	value.Destroy()
}

type LnOfferBlindedPath struct {
	BlindedHops []string
}

func (r *LnOfferBlindedPath) Destroy() {
	FfiDestroyerSequenceString{}.Destroy(r.BlindedHops)
}

type FfiConverterTypeLnOfferBlindedPath struct{}

var FfiConverterTypeLnOfferBlindedPathINSTANCE = FfiConverterTypeLnOfferBlindedPath{}

func (c FfiConverterTypeLnOfferBlindedPath) Lift(rb RustBufferI) LnOfferBlindedPath {
	return LiftFromRustBuffer[LnOfferBlindedPath](c, rb)
}

func (c FfiConverterTypeLnOfferBlindedPath) Read(reader io.Reader) LnOfferBlindedPath {
	return LnOfferBlindedPath{
		FfiConverterSequenceStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnOfferBlindedPath) Lower(value LnOfferBlindedPath) RustBuffer {
	return LowerIntoRustBuffer[LnOfferBlindedPath](c, value)
}

func (c FfiConverterTypeLnOfferBlindedPath) Write(writer io.Writer, value LnOfferBlindedPath) {
	FfiConverterSequenceStringINSTANCE.Write(writer, value.BlindedHops)
}

type FfiDestroyerTypeLnOfferBlindedPath struct{}

func (_ FfiDestroyerTypeLnOfferBlindedPath) Destroy(value LnOfferBlindedPath) {
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

type FfiConverterTypeLnUrlAuthRequestData struct{}

var FfiConverterTypeLnUrlAuthRequestDataINSTANCE = FfiConverterTypeLnUrlAuthRequestData{}

func (c FfiConverterTypeLnUrlAuthRequestData) Lift(rb RustBufferI) LnUrlAuthRequestData {
	return LiftFromRustBuffer[LnUrlAuthRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlAuthRequestData) Read(reader io.Reader) LnUrlAuthRequestData {
	return LnUrlAuthRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlAuthRequestData) Lower(value LnUrlAuthRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlAuthRequestData](c, value)
}

func (c FfiConverterTypeLnUrlAuthRequestData) Write(writer io.Writer, value LnUrlAuthRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Action)
}

type FfiDestroyerTypeLnUrlAuthRequestData struct{}

func (_ FfiDestroyerTypeLnUrlAuthRequestData) Destroy(value LnUrlAuthRequestData) {
	value.Destroy()
}

type LnUrlErrorData struct {
	Reason string
}

func (r *LnUrlErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterTypeLnUrlErrorData struct{}

var FfiConverterTypeLnUrlErrorDataINSTANCE = FfiConverterTypeLnUrlErrorData{}

func (c FfiConverterTypeLnUrlErrorData) Lift(rb RustBufferI) LnUrlErrorData {
	return LiftFromRustBuffer[LnUrlErrorData](c, rb)
}

func (c FfiConverterTypeLnUrlErrorData) Read(reader io.Reader) LnUrlErrorData {
	return LnUrlErrorData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlErrorData) Lower(value LnUrlErrorData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlErrorData](c, value)
}

func (c FfiConverterTypeLnUrlErrorData) Write(writer io.Writer, value LnUrlErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerTypeLnUrlErrorData struct{}

func (_ FfiDestroyerTypeLnUrlErrorData) Destroy(value LnUrlErrorData) {
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
	FfiDestroyerOptionalTypeSuccessActionProcessed{}.Destroy(r.LnurlPaySuccessAction)
	FfiDestroyerOptionalTypeSuccessAction{}.Destroy(r.LnurlPayUnprocessedSuccessAction)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlWithdrawEndpoint)
}

type FfiConverterTypeLnUrlInfo struct{}

var FfiConverterTypeLnUrlInfoINSTANCE = FfiConverterTypeLnUrlInfo{}

func (c FfiConverterTypeLnUrlInfo) Lift(rb RustBufferI) LnUrlInfo {
	return LiftFromRustBuffer[LnUrlInfo](c, rb)
}

func (c FfiConverterTypeLnUrlInfo) Read(reader io.Reader) LnUrlInfo {
	return LnUrlInfo{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterOptionalTypeSuccessActionINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlInfo) Lower(value LnUrlInfo) RustBuffer {
	return LowerIntoRustBuffer[LnUrlInfo](c, value)
}

func (c FfiConverterTypeLnUrlInfo) Write(writer io.Writer, value LnUrlInfo) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayComment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayDomain)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayMetadata)
	FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Write(writer, value.LnurlPaySuccessAction)
	FfiConverterOptionalTypeSuccessActionINSTANCE.Write(writer, value.LnurlPayUnprocessedSuccessAction)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlWithdrawEndpoint)
}

type FfiDestroyerTypeLnUrlInfo struct{}

func (_ FfiDestroyerTypeLnUrlInfo) Destroy(value LnUrlInfo) {
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

type FfiConverterTypeLnUrlPayErrorData struct{}

var FfiConverterTypeLnUrlPayErrorDataINSTANCE = FfiConverterTypeLnUrlPayErrorData{}

func (c FfiConverterTypeLnUrlPayErrorData) Lift(rb RustBufferI) LnUrlPayErrorData {
	return LiftFromRustBuffer[LnUrlPayErrorData](c, rb)
}

func (c FfiConverterTypeLnUrlPayErrorData) Read(reader io.Reader) LnUrlPayErrorData {
	return LnUrlPayErrorData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPayErrorData) Lower(value LnUrlPayErrorData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayErrorData](c, value)
}

func (c FfiConverterTypeLnUrlPayErrorData) Write(writer io.Writer, value LnUrlPayErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerTypeLnUrlPayErrorData struct{}

func (_ FfiDestroyerTypeLnUrlPayErrorData) Destroy(value LnUrlPayErrorData) {
	value.Destroy()
}

type LnUrlPayRequest struct {
	PrepareResponse PrepareLnUrlPayResponse
}

func (r *LnUrlPayRequest) Destroy() {
	FfiDestroyerTypePrepareLnUrlPayResponse{}.Destroy(r.PrepareResponse)
}

type FfiConverterTypeLnUrlPayRequest struct{}

var FfiConverterTypeLnUrlPayRequestINSTANCE = FfiConverterTypeLnUrlPayRequest{}

func (c FfiConverterTypeLnUrlPayRequest) Lift(rb RustBufferI) LnUrlPayRequest {
	return LiftFromRustBuffer[LnUrlPayRequest](c, rb)
}

func (c FfiConverterTypeLnUrlPayRequest) Read(reader io.Reader) LnUrlPayRequest {
	return LnUrlPayRequest{
		FfiConverterTypePrepareLnUrlPayResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPayRequest) Lower(value LnUrlPayRequest) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequest](c, value)
}

func (c FfiConverterTypeLnUrlPayRequest) Write(writer io.Writer, value LnUrlPayRequest) {
	FfiConverterTypePrepareLnUrlPayResponseINSTANCE.Write(writer, value.PrepareResponse)
}

type FfiDestroyerTypeLnUrlPayRequest struct{}

func (_ FfiDestroyerTypeLnUrlPayRequest) Destroy(value LnUrlPayRequest) {
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

type FfiConverterTypeLnUrlPayRequestData struct{}

var FfiConverterTypeLnUrlPayRequestDataINSTANCE = FfiConverterTypeLnUrlPayRequestData{}

func (c FfiConverterTypeLnUrlPayRequestData) Lift(rb RustBufferI) LnUrlPayRequestData {
	return LiftFromRustBuffer[LnUrlPayRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlPayRequestData) Read(reader io.Reader) LnUrlPayRequestData {
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

func (c FfiConverterTypeLnUrlPayRequestData) Lower(value LnUrlPayRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequestData](c, value)
}

func (c FfiConverterTypeLnUrlPayRequestData) Write(writer io.Writer, value LnUrlPayRequestData) {
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

type FfiDestroyerTypeLnUrlPayRequestData struct{}

func (_ FfiDestroyerTypeLnUrlPayRequestData) Destroy(value LnUrlPayRequestData) {
	value.Destroy()
}

type LnUrlPaySuccessData struct {
	SuccessAction *SuccessActionProcessed
	Payment       Payment
}

func (r *LnUrlPaySuccessData) Destroy() {
	FfiDestroyerOptionalTypeSuccessActionProcessed{}.Destroy(r.SuccessAction)
	FfiDestroyerTypePayment{}.Destroy(r.Payment)
}

type FfiConverterTypeLnUrlPaySuccessData struct{}

var FfiConverterTypeLnUrlPaySuccessDataINSTANCE = FfiConverterTypeLnUrlPaySuccessData{}

func (c FfiConverterTypeLnUrlPaySuccessData) Lift(rb RustBufferI) LnUrlPaySuccessData {
	return LiftFromRustBuffer[LnUrlPaySuccessData](c, rb)
}

func (c FfiConverterTypeLnUrlPaySuccessData) Read(reader io.Reader) LnUrlPaySuccessData {
	return LnUrlPaySuccessData{
		FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterTypePaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPaySuccessData) Lower(value LnUrlPaySuccessData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPaySuccessData](c, value)
}

func (c FfiConverterTypeLnUrlPaySuccessData) Write(writer io.Writer, value LnUrlPaySuccessData) {
	FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Write(writer, value.SuccessAction)
	FfiConverterTypePaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerTypeLnUrlPaySuccessData struct{}

func (_ FfiDestroyerTypeLnUrlPaySuccessData) Destroy(value LnUrlPaySuccessData) {
	value.Destroy()
}

type LnUrlWithdrawRequest struct {
	Data        LnUrlWithdrawRequestData
	AmountMsat  uint64
	Description *string
}

func (r *LnUrlWithdrawRequest) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
}

type FfiConverterTypeLnUrlWithdrawRequest struct{}

var FfiConverterTypeLnUrlWithdrawRequestINSTANCE = FfiConverterTypeLnUrlWithdrawRequest{}

func (c FfiConverterTypeLnUrlWithdrawRequest) Lift(rb RustBufferI) LnUrlWithdrawRequest {
	return LiftFromRustBuffer[LnUrlWithdrawRequest](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Read(reader io.Reader) LnUrlWithdrawRequest {
	return LnUrlWithdrawRequest{
		FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Lower(value LnUrlWithdrawRequest) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequest](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Write(writer io.Writer, value LnUrlWithdrawRequest) {
	FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
}

type FfiDestroyerTypeLnUrlWithdrawRequest struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawRequest) Destroy(value LnUrlWithdrawRequest) {
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

type FfiConverterTypeLnUrlWithdrawRequestData struct{}

var FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE = FfiConverterTypeLnUrlWithdrawRequestData{}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Lift(rb RustBufferI) LnUrlWithdrawRequestData {
	return LiftFromRustBuffer[LnUrlWithdrawRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Read(reader io.Reader) LnUrlWithdrawRequestData {
	return LnUrlWithdrawRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Lower(value LnUrlWithdrawRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequestData](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Write(writer io.Writer, value LnUrlWithdrawRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.DefaultDescription)
	FfiConverterUint64INSTANCE.Write(writer, value.MinWithdrawable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxWithdrawable)
}

type FfiDestroyerTypeLnUrlWithdrawRequestData struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawRequestData) Destroy(value LnUrlWithdrawRequestData) {
	value.Destroy()
}

type LnUrlWithdrawSuccessData struct {
	Invoice LnInvoice
}

func (r *LnUrlWithdrawSuccessData) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(r.Invoice)
}

type FfiConverterTypeLnUrlWithdrawSuccessData struct{}

var FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE = FfiConverterTypeLnUrlWithdrawSuccessData{}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Lift(rb RustBufferI) LnUrlWithdrawSuccessData {
	return LiftFromRustBuffer[LnUrlWithdrawSuccessData](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Read(reader io.Reader) LnUrlWithdrawSuccessData {
	return LnUrlWithdrawSuccessData{
		FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Lower(value LnUrlWithdrawSuccessData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawSuccessData](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Write(writer io.Writer, value LnUrlWithdrawSuccessData) {
	FfiConverterTypeLNInvoiceINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerTypeLnUrlWithdrawSuccessData struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawSuccessData) Destroy(value LnUrlWithdrawSuccessData) {
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
	FfiDestroyerTypeSymbol{}.Destroy(r.Symbol)
}

type FfiConverterTypeLocaleOverrides struct{}

var FfiConverterTypeLocaleOverridesINSTANCE = FfiConverterTypeLocaleOverrides{}

func (c FfiConverterTypeLocaleOverrides) Lift(rb RustBufferI) LocaleOverrides {
	return LiftFromRustBuffer[LocaleOverrides](c, rb)
}

func (c FfiConverterTypeLocaleOverrides) Read(reader io.Reader) LocaleOverrides {
	return LocaleOverrides{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterTypeSymbolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLocaleOverrides) Lower(value LocaleOverrides) RustBuffer {
	return LowerIntoRustBuffer[LocaleOverrides](c, value)
}

func (c FfiConverterTypeLocaleOverrides) Write(writer io.Writer, value LocaleOverrides) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterTypeSymbolINSTANCE.Write(writer, value.Symbol)
}

type FfiDestroyerTypeLocaleOverrides struct{}

func (_ FfiDestroyerTypeLocaleOverrides) Destroy(value LocaleOverrides) {
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

type FfiConverterTypeLocalizedName struct{}

var FfiConverterTypeLocalizedNameINSTANCE = FfiConverterTypeLocalizedName{}

func (c FfiConverterTypeLocalizedName) Lift(rb RustBufferI) LocalizedName {
	return LiftFromRustBuffer[LocalizedName](c, rb)
}

func (c FfiConverterTypeLocalizedName) Read(reader io.Reader) LocalizedName {
	return LocalizedName{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLocalizedName) Lower(value LocalizedName) RustBuffer {
	return LowerIntoRustBuffer[LocalizedName](c, value)
}

func (c FfiConverterTypeLocalizedName) Write(writer io.Writer, value LocalizedName) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
}

type FfiDestroyerTypeLocalizedName struct{}

func (_ FfiDestroyerTypeLocalizedName) Destroy(value LocalizedName) {
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

type FfiConverterTypeLogEntry struct{}

var FfiConverterTypeLogEntryINSTANCE = FfiConverterTypeLogEntry{}

func (c FfiConverterTypeLogEntry) Lift(rb RustBufferI) LogEntry {
	return LiftFromRustBuffer[LogEntry](c, rb)
}

func (c FfiConverterTypeLogEntry) Read(reader io.Reader) LogEntry {
	return LogEntry{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLogEntry) Lower(value LogEntry) RustBuffer {
	return LowerIntoRustBuffer[LogEntry](c, value)
}

func (c FfiConverterTypeLogEntry) Write(writer io.Writer, value LogEntry) {
	FfiConverterStringINSTANCE.Write(writer, value.Line)
	FfiConverterStringINSTANCE.Write(writer, value.Level)
}

type FfiDestroyerTypeLogEntry struct{}

func (_ FfiDestroyerTypeLogEntry) Destroy(value LogEntry) {
	value.Destroy()
}

type MessageSuccessActionData struct {
	Message string
}

func (r *MessageSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterTypeMessageSuccessActionData struct{}

var FfiConverterTypeMessageSuccessActionDataINSTANCE = FfiConverterTypeMessageSuccessActionData{}

func (c FfiConverterTypeMessageSuccessActionData) Lift(rb RustBufferI) MessageSuccessActionData {
	return LiftFromRustBuffer[MessageSuccessActionData](c, rb)
}

func (c FfiConverterTypeMessageSuccessActionData) Read(reader io.Reader) MessageSuccessActionData {
	return MessageSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMessageSuccessActionData) Lower(value MessageSuccessActionData) RustBuffer {
	return LowerIntoRustBuffer[MessageSuccessActionData](c, value)
}

func (c FfiConverterTypeMessageSuccessActionData) Write(writer io.Writer, value MessageSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeMessageSuccessActionData struct{}

func (_ FfiDestroyerTypeMessageSuccessActionData) Destroy(value MessageSuccessActionData) {
	value.Destroy()
}

type OnchainPaymentLimitsResponse struct {
	Send    Limits
	Receive Limits
}

func (r *OnchainPaymentLimitsResponse) Destroy() {
	FfiDestroyerTypeLimits{}.Destroy(r.Send)
	FfiDestroyerTypeLimits{}.Destroy(r.Receive)
}

type FfiConverterTypeOnchainPaymentLimitsResponse struct{}

var FfiConverterTypeOnchainPaymentLimitsResponseINSTANCE = FfiConverterTypeOnchainPaymentLimitsResponse{}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Lift(rb RustBufferI) OnchainPaymentLimitsResponse {
	return LiftFromRustBuffer[OnchainPaymentLimitsResponse](c, rb)
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Read(reader io.Reader) OnchainPaymentLimitsResponse {
	return OnchainPaymentLimitsResponse{
		FfiConverterTypeLimitsINSTANCE.Read(reader),
		FfiConverterTypeLimitsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Lower(value OnchainPaymentLimitsResponse) RustBuffer {
	return LowerIntoRustBuffer[OnchainPaymentLimitsResponse](c, value)
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Write(writer io.Writer, value OnchainPaymentLimitsResponse) {
	FfiConverterTypeLimitsINSTANCE.Write(writer, value.Send)
	FfiConverterTypeLimitsINSTANCE.Write(writer, value.Receive)
}

type FfiDestroyerTypeOnchainPaymentLimitsResponse struct{}

func (_ FfiDestroyerTypeOnchainPaymentLimitsResponse) Destroy(value OnchainPaymentLimitsResponse) {
	value.Destroy()
}

type PayOnchainRequest struct {
	Address         string
	PrepareResponse PreparePayOnchainResponse
}

func (r *PayOnchainRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerTypePreparePayOnchainResponse{}.Destroy(r.PrepareResponse)
}

type FfiConverterTypePayOnchainRequest struct{}

var FfiConverterTypePayOnchainRequestINSTANCE = FfiConverterTypePayOnchainRequest{}

func (c FfiConverterTypePayOnchainRequest) Lift(rb RustBufferI) PayOnchainRequest {
	return LiftFromRustBuffer[PayOnchainRequest](c, rb)
}

func (c FfiConverterTypePayOnchainRequest) Read(reader io.Reader) PayOnchainRequest {
	return PayOnchainRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypePreparePayOnchainResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayOnchainRequest) Lower(value PayOnchainRequest) RustBuffer {
	return LowerIntoRustBuffer[PayOnchainRequest](c, value)
}

func (c FfiConverterTypePayOnchainRequest) Write(writer io.Writer, value PayOnchainRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterTypePreparePayOnchainResponseINSTANCE.Write(writer, value.PrepareResponse)
}

type FfiDestroyerTypePayOnchainRequest struct{}

func (_ FfiDestroyerTypePayOnchainRequest) Destroy(value PayOnchainRequest) {
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
	FfiDestroyerTypePaymentType{}.Destroy(r.PaymentType)
	FfiDestroyerTypePaymentState{}.Destroy(r.Status)
	FfiDestroyerTypePaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalUint64{}.Destroy(r.SwapperFeesSat)
	FfiDestroyerOptionalString{}.Destroy(r.Destination)
	FfiDestroyerOptionalString{}.Destroy(r.TxId)
	FfiDestroyerOptionalString{}.Destroy(r.UnblindingData)
}

type FfiConverterTypePayment struct{}

var FfiConverterTypePaymentINSTANCE = FfiConverterTypePayment{}

func (c FfiConverterTypePayment) Lift(rb RustBufferI) Payment {
	return LiftFromRustBuffer[Payment](c, rb)
}

func (c FfiConverterTypePayment) Read(reader io.Reader) Payment {
	return Payment{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypePaymentTypeINSTANCE.Read(reader),
		FfiConverterTypePaymentStateINSTANCE.Read(reader),
		FfiConverterTypePaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayment) Lower(value Payment) RustBuffer {
	return LowerIntoRustBuffer[Payment](c, value)
}

func (c FfiConverterTypePayment) Write(writer io.Writer, value Payment) {
	FfiConverterUint32INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterTypePaymentTypeINSTANCE.Write(writer, value.PaymentType)
	FfiConverterTypePaymentStateINSTANCE.Write(writer, value.Status)
	FfiConverterTypePaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.SwapperFeesSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.TxId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.UnblindingData)
}

type FfiDestroyerTypePayment struct{}

func (_ FfiDestroyerTypePayment) Destroy(value Payment) {
	value.Destroy()
}

type PrepareBuyBitcoinRequest struct {
	Provider  BuyBitcoinProvider
	AmountSat uint64
}

func (r *PrepareBuyBitcoinRequest) Destroy() {
	FfiDestroyerTypeBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
}

type FfiConverterTypePrepareBuyBitcoinRequest struct{}

var FfiConverterTypePrepareBuyBitcoinRequestINSTANCE = FfiConverterTypePrepareBuyBitcoinRequest{}

func (c FfiConverterTypePrepareBuyBitcoinRequest) Lift(rb RustBufferI) PrepareBuyBitcoinRequest {
	return LiftFromRustBuffer[PrepareBuyBitcoinRequest](c, rb)
}

func (c FfiConverterTypePrepareBuyBitcoinRequest) Read(reader io.Reader) PrepareBuyBitcoinRequest {
	return PrepareBuyBitcoinRequest{
		FfiConverterTypeBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareBuyBitcoinRequest) Lower(value PrepareBuyBitcoinRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareBuyBitcoinRequest](c, value)
}

func (c FfiConverterTypePrepareBuyBitcoinRequest) Write(writer io.Writer, value PrepareBuyBitcoinRequest) {
	FfiConverterTypeBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
}

type FfiDestroyerTypePrepareBuyBitcoinRequest struct{}

func (_ FfiDestroyerTypePrepareBuyBitcoinRequest) Destroy(value PrepareBuyBitcoinRequest) {
	value.Destroy()
}

type PrepareBuyBitcoinResponse struct {
	Provider  BuyBitcoinProvider
	AmountSat uint64
	FeesSat   uint64
}

func (r *PrepareBuyBitcoinResponse) Destroy() {
	FfiDestroyerTypeBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
}

type FfiConverterTypePrepareBuyBitcoinResponse struct{}

var FfiConverterTypePrepareBuyBitcoinResponseINSTANCE = FfiConverterTypePrepareBuyBitcoinResponse{}

func (c FfiConverterTypePrepareBuyBitcoinResponse) Lift(rb RustBufferI) PrepareBuyBitcoinResponse {
	return LiftFromRustBuffer[PrepareBuyBitcoinResponse](c, rb)
}

func (c FfiConverterTypePrepareBuyBitcoinResponse) Read(reader io.Reader) PrepareBuyBitcoinResponse {
	return PrepareBuyBitcoinResponse{
		FfiConverterTypeBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareBuyBitcoinResponse) Lower(value PrepareBuyBitcoinResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareBuyBitcoinResponse](c, value)
}

func (c FfiConverterTypePrepareBuyBitcoinResponse) Write(writer io.Writer, value PrepareBuyBitcoinResponse) {
	FfiConverterTypeBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
}

type FfiDestroyerTypePrepareBuyBitcoinResponse struct{}

func (_ FfiDestroyerTypePrepareBuyBitcoinResponse) Destroy(value PrepareBuyBitcoinResponse) {
	value.Destroy()
}

type PrepareLnUrlPayRequest struct {
	Data                     LnUrlPayRequestData
	Amount                   PayAmount
	Comment                  *string
	ValidateSuccessActionUrl *bool
}

func (r *PrepareLnUrlPayRequest) Destroy() {
	FfiDestroyerTypeLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerTypePayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalBool{}.Destroy(r.ValidateSuccessActionUrl)
}

type FfiConverterTypePrepareLnUrlPayRequest struct{}

var FfiConverterTypePrepareLnUrlPayRequestINSTANCE = FfiConverterTypePrepareLnUrlPayRequest{}

func (c FfiConverterTypePrepareLnUrlPayRequest) Lift(rb RustBufferI) PrepareLnUrlPayRequest {
	return LiftFromRustBuffer[PrepareLnUrlPayRequest](c, rb)
}

func (c FfiConverterTypePrepareLnUrlPayRequest) Read(reader io.Reader) PrepareLnUrlPayRequest {
	return PrepareLnUrlPayRequest{
		FfiConverterTypeLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterTypePayAmountINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareLnUrlPayRequest) Lower(value PrepareLnUrlPayRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareLnUrlPayRequest](c, value)
}

func (c FfiConverterTypePrepareLnUrlPayRequest) Write(writer io.Writer, value PrepareLnUrlPayRequest) {
	FfiConverterTypeLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterTypePayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.ValidateSuccessActionUrl)
}

type FfiDestroyerTypePrepareLnUrlPayRequest struct{}

func (_ FfiDestroyerTypePrepareLnUrlPayRequest) Destroy(value PrepareLnUrlPayRequest) {
	value.Destroy()
}

type PrepareLnUrlPayResponse struct {
	Destination   SendDestination
	FeesSat       uint64
	Data          LnUrlPayRequestData
	Comment       *string
	SuccessAction *SuccessAction
}

func (r *PrepareLnUrlPayResponse) Destroy() {
	FfiDestroyerTypeSendDestination{}.Destroy(r.Destination)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerTypeLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalTypeSuccessAction{}.Destroy(r.SuccessAction)
}

type FfiConverterTypePrepareLnUrlPayResponse struct{}

var FfiConverterTypePrepareLnUrlPayResponseINSTANCE = FfiConverterTypePrepareLnUrlPayResponse{}

func (c FfiConverterTypePrepareLnUrlPayResponse) Lift(rb RustBufferI) PrepareLnUrlPayResponse {
	return LiftFromRustBuffer[PrepareLnUrlPayResponse](c, rb)
}

func (c FfiConverterTypePrepareLnUrlPayResponse) Read(reader io.Reader) PrepareLnUrlPayResponse {
	return PrepareLnUrlPayResponse{
		FfiConverterTypeSendDestinationINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypeLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeSuccessActionINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareLnUrlPayResponse) Lower(value PrepareLnUrlPayResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareLnUrlPayResponse](c, value)
}

func (c FfiConverterTypePrepareLnUrlPayResponse) Write(writer io.Writer, value PrepareLnUrlPayResponse) {
	FfiConverterTypeSendDestinationINSTANCE.Write(writer, value.Destination)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterTypeLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalTypeSuccessActionINSTANCE.Write(writer, value.SuccessAction)
}

type FfiDestroyerTypePrepareLnUrlPayResponse struct{}

func (_ FfiDestroyerTypePrepareLnUrlPayResponse) Destroy(value PrepareLnUrlPayResponse) {
	value.Destroy()
}

type PreparePayOnchainRequest struct {
	Amount             PayAmount
	FeeRateSatPerVbyte *uint32
}

func (r *PreparePayOnchainRequest) Destroy() {
	FfiDestroyerTypePayAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint32{}.Destroy(r.FeeRateSatPerVbyte)
}

type FfiConverterTypePreparePayOnchainRequest struct{}

var FfiConverterTypePreparePayOnchainRequestINSTANCE = FfiConverterTypePreparePayOnchainRequest{}

func (c FfiConverterTypePreparePayOnchainRequest) Lift(rb RustBufferI) PreparePayOnchainRequest {
	return LiftFromRustBuffer[PreparePayOnchainRequest](c, rb)
}

func (c FfiConverterTypePreparePayOnchainRequest) Read(reader io.Reader) PreparePayOnchainRequest {
	return PreparePayOnchainRequest{
		FfiConverterTypePayAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePreparePayOnchainRequest) Lower(value PreparePayOnchainRequest) RustBuffer {
	return LowerIntoRustBuffer[PreparePayOnchainRequest](c, value)
}

func (c FfiConverterTypePreparePayOnchainRequest) Write(writer io.Writer, value PreparePayOnchainRequest) {
	FfiConverterTypePayAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerTypePreparePayOnchainRequest struct{}

func (_ FfiDestroyerTypePreparePayOnchainRequest) Destroy(value PreparePayOnchainRequest) {
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

type FfiConverterTypePreparePayOnchainResponse struct{}

var FfiConverterTypePreparePayOnchainResponseINSTANCE = FfiConverterTypePreparePayOnchainResponse{}

func (c FfiConverterTypePreparePayOnchainResponse) Lift(rb RustBufferI) PreparePayOnchainResponse {
	return LiftFromRustBuffer[PreparePayOnchainResponse](c, rb)
}

func (c FfiConverterTypePreparePayOnchainResponse) Read(reader io.Reader) PreparePayOnchainResponse {
	return PreparePayOnchainResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePreparePayOnchainResponse) Lower(value PreparePayOnchainResponse) RustBuffer {
	return LowerIntoRustBuffer[PreparePayOnchainResponse](c, value)
}

func (c FfiConverterTypePreparePayOnchainResponse) Write(writer io.Writer, value PreparePayOnchainResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.ReceiverAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.ClaimFeesSat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalFeesSat)
}

type FfiDestroyerTypePreparePayOnchainResponse struct{}

func (_ FfiDestroyerTypePreparePayOnchainResponse) Destroy(value PreparePayOnchainResponse) {
	value.Destroy()
}

type PrepareReceiveRequest struct {
	PaymentMethod PaymentMethod
	Amount        *ReceiveAmount
}

func (r *PrepareReceiveRequest) Destroy() {
	FfiDestroyerTypePaymentMethod{}.Destroy(r.PaymentMethod)
	FfiDestroyerOptionalTypeReceiveAmount{}.Destroy(r.Amount)
}

type FfiConverterTypePrepareReceiveRequest struct{}

var FfiConverterTypePrepareReceiveRequestINSTANCE = FfiConverterTypePrepareReceiveRequest{}

func (c FfiConverterTypePrepareReceiveRequest) Lift(rb RustBufferI) PrepareReceiveRequest {
	return LiftFromRustBuffer[PrepareReceiveRequest](c, rb)
}

func (c FfiConverterTypePrepareReceiveRequest) Read(reader io.Reader) PrepareReceiveRequest {
	return PrepareReceiveRequest{
		FfiConverterTypePaymentMethodINSTANCE.Read(reader),
		FfiConverterOptionalTypeReceiveAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareReceiveRequest) Lower(value PrepareReceiveRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveRequest](c, value)
}

func (c FfiConverterTypePrepareReceiveRequest) Write(writer io.Writer, value PrepareReceiveRequest) {
	FfiConverterTypePaymentMethodINSTANCE.Write(writer, value.PaymentMethod)
	FfiConverterOptionalTypeReceiveAmountINSTANCE.Write(writer, value.Amount)
}

type FfiDestroyerTypePrepareReceiveRequest struct{}

func (_ FfiDestroyerTypePrepareReceiveRequest) Destroy(value PrepareReceiveRequest) {
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
	FfiDestroyerTypePaymentMethod{}.Destroy(r.PaymentMethod)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
	FfiDestroyerOptionalTypeReceiveAmount{}.Destroy(r.Amount)
	FfiDestroyerOptionalUint64{}.Destroy(r.MinPayerAmountSat)
	FfiDestroyerOptionalUint64{}.Destroy(r.MaxPayerAmountSat)
	FfiDestroyerOptionalFloat64{}.Destroy(r.SwapperFeerate)
}

type FfiConverterTypePrepareReceiveResponse struct{}

var FfiConverterTypePrepareReceiveResponseINSTANCE = FfiConverterTypePrepareReceiveResponse{}

func (c FfiConverterTypePrepareReceiveResponse) Lift(rb RustBufferI) PrepareReceiveResponse {
	return LiftFromRustBuffer[PrepareReceiveResponse](c, rb)
}

func (c FfiConverterTypePrepareReceiveResponse) Read(reader io.Reader) PrepareReceiveResponse {
	return PrepareReceiveResponse{
		FfiConverterTypePaymentMethodINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalTypeReceiveAmountINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareReceiveResponse) Lower(value PrepareReceiveResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveResponse](c, value)
}

func (c FfiConverterTypePrepareReceiveResponse) Write(writer io.Writer, value PrepareReceiveResponse) {
	FfiConverterTypePaymentMethodINSTANCE.Write(writer, value.PaymentMethod)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
	FfiConverterOptionalTypeReceiveAmountINSTANCE.Write(writer, value.Amount)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.MinPayerAmountSat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.MaxPayerAmountSat)
	FfiConverterOptionalFloat64INSTANCE.Write(writer, value.SwapperFeerate)
}

type FfiDestroyerTypePrepareReceiveResponse struct{}

func (_ FfiDestroyerTypePrepareReceiveResponse) Destroy(value PrepareReceiveResponse) {
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

type FfiConverterTypePrepareRefundRequest struct{}

var FfiConverterTypePrepareRefundRequestINSTANCE = FfiConverterTypePrepareRefundRequest{}

func (c FfiConverterTypePrepareRefundRequest) Lift(rb RustBufferI) PrepareRefundRequest {
	return LiftFromRustBuffer[PrepareRefundRequest](c, rb)
}

func (c FfiConverterTypePrepareRefundRequest) Read(reader io.Reader) PrepareRefundRequest {
	return PrepareRefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRefundRequest) Lower(value PrepareRefundRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundRequest](c, value)
}

func (c FfiConverterTypePrepareRefundRequest) Write(writer io.Writer, value PrepareRefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.RefundAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerTypePrepareRefundRequest struct{}

func (_ FfiDestroyerTypePrepareRefundRequest) Destroy(value PrepareRefundRequest) {
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

type FfiConverterTypePrepareRefundResponse struct{}

var FfiConverterTypePrepareRefundResponseINSTANCE = FfiConverterTypePrepareRefundResponse{}

func (c FfiConverterTypePrepareRefundResponse) Lift(rb RustBufferI) PrepareRefundResponse {
	return LiftFromRustBuffer[PrepareRefundResponse](c, rb)
}

func (c FfiConverterTypePrepareRefundResponse) Read(reader io.Reader) PrepareRefundResponse {
	return PrepareRefundResponse{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRefundResponse) Lower(value PrepareRefundResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundResponse](c, value)
}

func (c FfiConverterTypePrepareRefundResponse) Write(writer io.Writer, value PrepareRefundResponse) {
	FfiConverterUint32INSTANCE.Write(writer, value.TxVsize)
	FfiConverterUint64INSTANCE.Write(writer, value.TxFeeSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRefundTxId)
}

type FfiDestroyerTypePrepareRefundResponse struct{}

func (_ FfiDestroyerTypePrepareRefundResponse) Destroy(value PrepareRefundResponse) {
	value.Destroy()
}

type PrepareSendRequest struct {
	Destination string
	Amount      *PayAmount
}

func (r *PrepareSendRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Destination)
	FfiDestroyerOptionalTypePayAmount{}.Destroy(r.Amount)
}

type FfiConverterTypePrepareSendRequest struct{}

var FfiConverterTypePrepareSendRequestINSTANCE = FfiConverterTypePrepareSendRequest{}

func (c FfiConverterTypePrepareSendRequest) Lift(rb RustBufferI) PrepareSendRequest {
	return LiftFromRustBuffer[PrepareSendRequest](c, rb)
}

func (c FfiConverterTypePrepareSendRequest) Read(reader io.Reader) PrepareSendRequest {
	return PrepareSendRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalTypePayAmountINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareSendRequest) Lower(value PrepareSendRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareSendRequest](c, value)
}

func (c FfiConverterTypePrepareSendRequest) Write(writer io.Writer, value PrepareSendRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Destination)
	FfiConverterOptionalTypePayAmountINSTANCE.Write(writer, value.Amount)
}

type FfiDestroyerTypePrepareSendRequest struct{}

func (_ FfiDestroyerTypePrepareSendRequest) Destroy(value PrepareSendRequest) {
	value.Destroy()
}

type PrepareSendResponse struct {
	Destination SendDestination
	FeesSat     uint64
}

func (r *PrepareSendResponse) Destroy() {
	FfiDestroyerTypeSendDestination{}.Destroy(r.Destination)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
}

type FfiConverterTypePrepareSendResponse struct{}

var FfiConverterTypePrepareSendResponseINSTANCE = FfiConverterTypePrepareSendResponse{}

func (c FfiConverterTypePrepareSendResponse) Lift(rb RustBufferI) PrepareSendResponse {
	return LiftFromRustBuffer[PrepareSendResponse](c, rb)
}

func (c FfiConverterTypePrepareSendResponse) Read(reader io.Reader) PrepareSendResponse {
	return PrepareSendResponse{
		FfiConverterTypeSendDestinationINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareSendResponse) Lower(value PrepareSendResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareSendResponse](c, value)
}

func (c FfiConverterTypePrepareSendResponse) Write(writer io.Writer, value PrepareSendResponse) {
	FfiConverterTypeSendDestinationINSTANCE.Write(writer, value.Destination)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
}

type FfiDestroyerTypePrepareSendResponse struct{}

func (_ FfiDestroyerTypePrepareSendResponse) Destroy(value PrepareSendResponse) {
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

type FfiConverterTypeRate struct{}

var FfiConverterTypeRateINSTANCE = FfiConverterTypeRate{}

func (c FfiConverterTypeRate) Lift(rb RustBufferI) Rate {
	return LiftFromRustBuffer[Rate](c, rb)
}

func (c FfiConverterTypeRate) Read(reader io.Reader) Rate {
	return Rate{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRate) Lower(value Rate) RustBuffer {
	return LowerIntoRustBuffer[Rate](c, value)
}

func (c FfiConverterTypeRate) Write(writer io.Writer, value Rate) {
	FfiConverterStringINSTANCE.Write(writer, value.Coin)
	FfiConverterFloat64INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerTypeRate struct{}

func (_ FfiDestroyerTypeRate) Destroy(value Rate) {
	value.Destroy()
}

type ReceivePaymentRequest struct {
	PrepareResponse    PrepareReceiveResponse
	Description        *string
	UseDescriptionHash *bool
}

func (r *ReceivePaymentRequest) Destroy() {
	FfiDestroyerTypePrepareReceiveResponse{}.Destroy(r.PrepareResponse)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalBool{}.Destroy(r.UseDescriptionHash)
}

type FfiConverterTypeReceivePaymentRequest struct{}

var FfiConverterTypeReceivePaymentRequestINSTANCE = FfiConverterTypeReceivePaymentRequest{}

func (c FfiConverterTypeReceivePaymentRequest) Lift(rb RustBufferI) ReceivePaymentRequest {
	return LiftFromRustBuffer[ReceivePaymentRequest](c, rb)
}

func (c FfiConverterTypeReceivePaymentRequest) Read(reader io.Reader) ReceivePaymentRequest {
	return ReceivePaymentRequest{
		FfiConverterTypePrepareReceiveResponseINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceivePaymentRequest) Lower(value ReceivePaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentRequest](c, value)
}

func (c FfiConverterTypeReceivePaymentRequest) Write(writer io.Writer, value ReceivePaymentRequest) {
	FfiConverterTypePrepareReceiveResponseINSTANCE.Write(writer, value.PrepareResponse)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.UseDescriptionHash)
}

type FfiDestroyerTypeReceivePaymentRequest struct{}

func (_ FfiDestroyerTypeReceivePaymentRequest) Destroy(value ReceivePaymentRequest) {
	value.Destroy()
}

type ReceivePaymentResponse struct {
	Destination string
}

func (r *ReceivePaymentResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Destination)
}

type FfiConverterTypeReceivePaymentResponse struct{}

var FfiConverterTypeReceivePaymentResponseINSTANCE = FfiConverterTypeReceivePaymentResponse{}

func (c FfiConverterTypeReceivePaymentResponse) Lift(rb RustBufferI) ReceivePaymentResponse {
	return LiftFromRustBuffer[ReceivePaymentResponse](c, rb)
}

func (c FfiConverterTypeReceivePaymentResponse) Read(reader io.Reader) ReceivePaymentResponse {
	return ReceivePaymentResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceivePaymentResponse) Lower(value ReceivePaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentResponse](c, value)
}

func (c FfiConverterTypeReceivePaymentResponse) Write(writer io.Writer, value ReceivePaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Destination)
}

type FfiDestroyerTypeReceivePaymentResponse struct{}

func (_ FfiDestroyerTypeReceivePaymentResponse) Destroy(value ReceivePaymentResponse) {
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

type FfiConverterTypeRecommendedFees struct{}

var FfiConverterTypeRecommendedFeesINSTANCE = FfiConverterTypeRecommendedFees{}

func (c FfiConverterTypeRecommendedFees) Lift(rb RustBufferI) RecommendedFees {
	return LiftFromRustBuffer[RecommendedFees](c, rb)
}

func (c FfiConverterTypeRecommendedFees) Read(reader io.Reader) RecommendedFees {
	return RecommendedFees{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRecommendedFees) Lower(value RecommendedFees) RustBuffer {
	return LowerIntoRustBuffer[RecommendedFees](c, value)
}

func (c FfiConverterTypeRecommendedFees) Write(writer io.Writer, value RecommendedFees) {
	FfiConverterUint64INSTANCE.Write(writer, value.FastestFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HalfHourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.EconomyFee)
	FfiConverterUint64INSTANCE.Write(writer, value.MinimumFee)
}

type FfiDestroyerTypeRecommendedFees struct{}

func (_ FfiDestroyerTypeRecommendedFees) Destroy(value RecommendedFees) {
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

type FfiConverterTypeRefundRequest struct{}

var FfiConverterTypeRefundRequestINSTANCE = FfiConverterTypeRefundRequest{}

func (c FfiConverterTypeRefundRequest) Lift(rb RustBufferI) RefundRequest {
	return LiftFromRustBuffer[RefundRequest](c, rb)
}

func (c FfiConverterTypeRefundRequest) Read(reader io.Reader) RefundRequest {
	return RefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRefundRequest) Lower(value RefundRequest) RustBuffer {
	return LowerIntoRustBuffer[RefundRequest](c, value)
}

func (c FfiConverterTypeRefundRequest) Write(writer io.Writer, value RefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.RefundAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.FeeRateSatPerVbyte)
}

type FfiDestroyerTypeRefundRequest struct{}

func (_ FfiDestroyerTypeRefundRequest) Destroy(value RefundRequest) {
	value.Destroy()
}

type RefundResponse struct {
	RefundTxId string
}

func (r *RefundResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.RefundTxId)
}

type FfiConverterTypeRefundResponse struct{}

var FfiConverterTypeRefundResponseINSTANCE = FfiConverterTypeRefundResponse{}

func (c FfiConverterTypeRefundResponse) Lift(rb RustBufferI) RefundResponse {
	return LiftFromRustBuffer[RefundResponse](c, rb)
}

func (c FfiConverterTypeRefundResponse) Read(reader io.Reader) RefundResponse {
	return RefundResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRefundResponse) Lower(value RefundResponse) RustBuffer {
	return LowerIntoRustBuffer[RefundResponse](c, value)
}

func (c FfiConverterTypeRefundResponse) Write(writer io.Writer, value RefundResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.RefundTxId)
}

type FfiDestroyerTypeRefundResponse struct{}

func (_ FfiDestroyerTypeRefundResponse) Destroy(value RefundResponse) {
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

type FfiConverterTypeRefundableSwap struct{}

var FfiConverterTypeRefundableSwapINSTANCE = FfiConverterTypeRefundableSwap{}

func (c FfiConverterTypeRefundableSwap) Lift(rb RustBufferI) RefundableSwap {
	return LiftFromRustBuffer[RefundableSwap](c, rb)
}

func (c FfiConverterTypeRefundableSwap) Read(reader io.Reader) RefundableSwap {
	return RefundableSwap{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRefundableSwap) Lower(value RefundableSwap) RustBuffer {
	return LowerIntoRustBuffer[RefundableSwap](c, value)
}

func (c FfiConverterTypeRefundableSwap) Write(writer io.Writer, value RefundableSwap) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRefundTxId)
}

type FfiDestroyerTypeRefundableSwap struct{}

func (_ FfiDestroyerTypeRefundableSwap) Destroy(value RefundableSwap) {
	value.Destroy()
}

type RestoreRequest struct {
	BackupPath *string
}

func (r *RestoreRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.BackupPath)
}

type FfiConverterTypeRestoreRequest struct{}

var FfiConverterTypeRestoreRequestINSTANCE = FfiConverterTypeRestoreRequest{}

func (c FfiConverterTypeRestoreRequest) Lift(rb RustBufferI) RestoreRequest {
	return LiftFromRustBuffer[RestoreRequest](c, rb)
}

func (c FfiConverterTypeRestoreRequest) Read(reader io.Reader) RestoreRequest {
	return RestoreRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRestoreRequest) Lower(value RestoreRequest) RustBuffer {
	return LowerIntoRustBuffer[RestoreRequest](c, value)
}

func (c FfiConverterTypeRestoreRequest) Write(writer io.Writer, value RestoreRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.BackupPath)
}

type FfiDestroyerTypeRestoreRequest struct{}

func (_ FfiDestroyerTypeRestoreRequest) Destroy(value RestoreRequest) {
	value.Destroy()
}

type RouteHint struct {
	Hops []RouteHintHop
}

func (r *RouteHint) Destroy() {
	FfiDestroyerSequenceTypeRouteHintHop{}.Destroy(r.Hops)
}

type FfiConverterTypeRouteHint struct{}

var FfiConverterTypeRouteHintINSTANCE = FfiConverterTypeRouteHint{}

func (c FfiConverterTypeRouteHint) Lift(rb RustBufferI) RouteHint {
	return LiftFromRustBuffer[RouteHint](c, rb)
}

func (c FfiConverterTypeRouteHint) Read(reader io.Reader) RouteHint {
	return RouteHint{
		FfiConverterSequenceTypeRouteHintHopINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRouteHint) Lower(value RouteHint) RustBuffer {
	return LowerIntoRustBuffer[RouteHint](c, value)
}

func (c FfiConverterTypeRouteHint) Write(writer io.Writer, value RouteHint) {
	FfiConverterSequenceTypeRouteHintHopINSTANCE.Write(writer, value.Hops)
}

type FfiDestroyerTypeRouteHint struct{}

func (_ FfiDestroyerTypeRouteHint) Destroy(value RouteHint) {
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

type FfiConverterTypeRouteHintHop struct{}

var FfiConverterTypeRouteHintHopINSTANCE = FfiConverterTypeRouteHintHop{}

func (c FfiConverterTypeRouteHintHop) Lift(rb RustBufferI) RouteHintHop {
	return LiftFromRustBuffer[RouteHintHop](c, rb)
}

func (c FfiConverterTypeRouteHintHop) Read(reader io.Reader) RouteHintHop {
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

func (c FfiConverterTypeRouteHintHop) Lower(value RouteHintHop) RustBuffer {
	return LowerIntoRustBuffer[RouteHintHop](c, value)
}

func (c FfiConverterTypeRouteHintHop) Write(writer io.Writer, value RouteHintHop) {
	FfiConverterStringINSTANCE.Write(writer, value.SrcNodeId)
	FfiConverterStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesBaseMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesProportionalMillionths)
	FfiConverterUint64INSTANCE.Write(writer, value.CltvExpiryDelta)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMinimumMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMaximumMsat)
}

type FfiDestroyerTypeRouteHintHop struct{}

func (_ FfiDestroyerTypeRouteHintHop) Destroy(value RouteHintHop) {
	value.Destroy()
}

type SendPaymentRequest struct {
	PrepareResponse PrepareSendResponse
}

func (r *SendPaymentRequest) Destroy() {
	FfiDestroyerTypePrepareSendResponse{}.Destroy(r.PrepareResponse)
}

type FfiConverterTypeSendPaymentRequest struct{}

var FfiConverterTypeSendPaymentRequestINSTANCE = FfiConverterTypeSendPaymentRequest{}

func (c FfiConverterTypeSendPaymentRequest) Lift(rb RustBufferI) SendPaymentRequest {
	return LiftFromRustBuffer[SendPaymentRequest](c, rb)
}

func (c FfiConverterTypeSendPaymentRequest) Read(reader io.Reader) SendPaymentRequest {
	return SendPaymentRequest{
		FfiConverterTypePrepareSendResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendPaymentRequest) Lower(value SendPaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[SendPaymentRequest](c, value)
}

func (c FfiConverterTypeSendPaymentRequest) Write(writer io.Writer, value SendPaymentRequest) {
	FfiConverterTypePrepareSendResponseINSTANCE.Write(writer, value.PrepareResponse)
}

type FfiDestroyerTypeSendPaymentRequest struct{}

func (_ FfiDestroyerTypeSendPaymentRequest) Destroy(value SendPaymentRequest) {
	value.Destroy()
}

type SendPaymentResponse struct {
	Payment Payment
}

func (r *SendPaymentResponse) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(r.Payment)
}

type FfiConverterTypeSendPaymentResponse struct{}

var FfiConverterTypeSendPaymentResponseINSTANCE = FfiConverterTypeSendPaymentResponse{}

func (c FfiConverterTypeSendPaymentResponse) Lift(rb RustBufferI) SendPaymentResponse {
	return LiftFromRustBuffer[SendPaymentResponse](c, rb)
}

func (c FfiConverterTypeSendPaymentResponse) Read(reader io.Reader) SendPaymentResponse {
	return SendPaymentResponse{
		FfiConverterTypePaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendPaymentResponse) Lower(value SendPaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[SendPaymentResponse](c, value)
}

func (c FfiConverterTypeSendPaymentResponse) Write(writer io.Writer, value SendPaymentResponse) {
	FfiConverterTypePaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerTypeSendPaymentResponse struct{}

func (_ FfiDestroyerTypeSendPaymentResponse) Destroy(value SendPaymentResponse) {
	value.Destroy()
}

type SignMessageRequest struct {
	Message string
}

func (r *SignMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterTypeSignMessageRequest struct{}

var FfiConverterTypeSignMessageRequestINSTANCE = FfiConverterTypeSignMessageRequest{}

func (c FfiConverterTypeSignMessageRequest) Lift(rb RustBufferI) SignMessageRequest {
	return LiftFromRustBuffer[SignMessageRequest](c, rb)
}

func (c FfiConverterTypeSignMessageRequest) Read(reader io.Reader) SignMessageRequest {
	return SignMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSignMessageRequest) Lower(value SignMessageRequest) RustBuffer {
	return LowerIntoRustBuffer[SignMessageRequest](c, value)
}

func (c FfiConverterTypeSignMessageRequest) Write(writer io.Writer, value SignMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeSignMessageRequest struct{}

func (_ FfiDestroyerTypeSignMessageRequest) Destroy(value SignMessageRequest) {
	value.Destroy()
}

type SignMessageResponse struct {
	Signature string
}

func (r *SignMessageResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterTypeSignMessageResponse struct{}

var FfiConverterTypeSignMessageResponseINSTANCE = FfiConverterTypeSignMessageResponse{}

func (c FfiConverterTypeSignMessageResponse) Lift(rb RustBufferI) SignMessageResponse {
	return LiftFromRustBuffer[SignMessageResponse](c, rb)
}

func (c FfiConverterTypeSignMessageResponse) Read(reader io.Reader) SignMessageResponse {
	return SignMessageResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSignMessageResponse) Lower(value SignMessageResponse) RustBuffer {
	return LowerIntoRustBuffer[SignMessageResponse](c, value)
}

func (c FfiConverterTypeSignMessageResponse) Write(writer io.Writer, value SignMessageResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerTypeSignMessageResponse struct{}

func (_ FfiDestroyerTypeSignMessageResponse) Destroy(value SignMessageResponse) {
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

type FfiConverterTypeSymbol struct{}

var FfiConverterTypeSymbolINSTANCE = FfiConverterTypeSymbol{}

func (c FfiConverterTypeSymbol) Lift(rb RustBufferI) Symbol {
	return LiftFromRustBuffer[Symbol](c, rb)
}

func (c FfiConverterTypeSymbol) Read(reader io.Reader) Symbol {
	return Symbol{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSymbol) Lower(value Symbol) RustBuffer {
	return LowerIntoRustBuffer[Symbol](c, value)
}

func (c FfiConverterTypeSymbol) Write(writer io.Writer, value Symbol) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Grapheme)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Template)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Rtl)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Position)
}

type FfiDestroyerTypeSymbol struct{}

func (_ FfiDestroyerTypeSymbol) Destroy(value Symbol) {
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

type FfiConverterTypeUrlSuccessActionData struct{}

var FfiConverterTypeUrlSuccessActionDataINSTANCE = FfiConverterTypeUrlSuccessActionData{}

func (c FfiConverterTypeUrlSuccessActionData) Lift(rb RustBufferI) UrlSuccessActionData {
	return LiftFromRustBuffer[UrlSuccessActionData](c, rb)
}

func (c FfiConverterTypeUrlSuccessActionData) Read(reader io.Reader) UrlSuccessActionData {
	return UrlSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeUrlSuccessActionData) Lower(value UrlSuccessActionData) RustBuffer {
	return LowerIntoRustBuffer[UrlSuccessActionData](c, value)
}

func (c FfiConverterTypeUrlSuccessActionData) Write(writer io.Writer, value UrlSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterBoolINSTANCE.Write(writer, value.MatchesCallbackDomain)
}

type FfiDestroyerTypeUrlSuccessActionData struct{}

func (_ FfiDestroyerTypeUrlSuccessActionData) Destroy(value UrlSuccessActionData) {
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
	FfiDestroyerSequenceTypeAssetBalance{}.Destroy(r.AssetBalances)
}

type FfiConverterTypeWalletInfo struct{}

var FfiConverterTypeWalletInfoINSTANCE = FfiConverterTypeWalletInfo{}

func (c FfiConverterTypeWalletInfo) Lift(rb RustBufferI) WalletInfo {
	return LiftFromRustBuffer[WalletInfo](c, rb)
}

func (c FfiConverterTypeWalletInfo) Read(reader io.Reader) WalletInfo {
	return WalletInfo{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterSequenceTypeAssetBalanceINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeWalletInfo) Lower(value WalletInfo) RustBuffer {
	return LowerIntoRustBuffer[WalletInfo](c, value)
}

func (c FfiConverterTypeWalletInfo) Write(writer io.Writer, value WalletInfo) {
	FfiConverterUint64INSTANCE.Write(writer, value.BalanceSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingSendSat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingReceiveSat)
	FfiConverterStringINSTANCE.Write(writer, value.Fingerprint)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterSequenceTypeAssetBalanceINSTANCE.Write(writer, value.AssetBalances)
}

type FfiDestroyerTypeWalletInfo struct{}

func (_ FfiDestroyerTypeWalletInfo) Destroy(value WalletInfo) {
	value.Destroy()
}

type AesSuccessActionDataResult interface {
	Destroy()
}
type AesSuccessActionDataResultDecrypted struct {
	Data AesSuccessActionDataDecrypted
}

func (e AesSuccessActionDataResultDecrypted) Destroy() {
	FfiDestroyerTypeAesSuccessActionDataDecrypted{}.Destroy(e.Data)
}

type AesSuccessActionDataResultErrorStatus struct {
	Reason string
}

func (e AesSuccessActionDataResultErrorStatus) Destroy() {
	FfiDestroyerString{}.Destroy(e.Reason)
}

type FfiConverterTypeAesSuccessActionDataResult struct{}

var FfiConverterTypeAesSuccessActionDataResultINSTANCE = FfiConverterTypeAesSuccessActionDataResult{}

func (c FfiConverterTypeAesSuccessActionDataResult) Lift(rb RustBufferI) AesSuccessActionDataResult {
	return LiftFromRustBuffer[AesSuccessActionDataResult](c, rb)
}

func (c FfiConverterTypeAesSuccessActionDataResult) Lower(value AesSuccessActionDataResult) RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataResult](c, value)
}
func (FfiConverterTypeAesSuccessActionDataResult) Read(reader io.Reader) AesSuccessActionDataResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return AesSuccessActionDataResultDecrypted{
			FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE.Read(reader),
		}
	case 2:
		return AesSuccessActionDataResultErrorStatus{
			FfiConverterStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeAesSuccessActionDataResult.Read()", id))
	}
}

func (FfiConverterTypeAesSuccessActionDataResult) Write(writer io.Writer, value AesSuccessActionDataResult) {
	switch variant_value := value.(type) {
	case AesSuccessActionDataResultDecrypted:
		writeInt32(writer, 1)
		FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE.Write(writer, variant_value.Data)
	case AesSuccessActionDataResultErrorStatus:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Reason)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeAesSuccessActionDataResult.Write", value))
	}
}

type FfiDestroyerTypeAesSuccessActionDataResult struct{}

func (_ FfiDestroyerTypeAesSuccessActionDataResult) Destroy(value AesSuccessActionDataResult) {
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

type FfiConverterTypeAmount struct{}

var FfiConverterTypeAmountINSTANCE = FfiConverterTypeAmount{}

func (c FfiConverterTypeAmount) Lift(rb RustBufferI) Amount {
	return LiftFromRustBuffer[Amount](c, rb)
}

func (c FfiConverterTypeAmount) Lower(value Amount) RustBuffer {
	return LowerIntoRustBuffer[Amount](c, value)
}
func (FfiConverterTypeAmount) Read(reader io.Reader) Amount {
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
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeAmount.Read()", id))
	}
}

func (FfiConverterTypeAmount) Write(writer io.Writer, value Amount) {
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
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeAmount.Write", value))
	}
}

type FfiDestroyerTypeAmount struct{}

func (_ FfiDestroyerTypeAmount) Destroy(value Amount) {
	value.Destroy()
}

type BuyBitcoinProvider uint

const (
	BuyBitcoinProviderMoonpay BuyBitcoinProvider = 1
)

type FfiConverterTypeBuyBitcoinProvider struct{}

var FfiConverterTypeBuyBitcoinProviderINSTANCE = FfiConverterTypeBuyBitcoinProvider{}

func (c FfiConverterTypeBuyBitcoinProvider) Lift(rb RustBufferI) BuyBitcoinProvider {
	return LiftFromRustBuffer[BuyBitcoinProvider](c, rb)
}

func (c FfiConverterTypeBuyBitcoinProvider) Lower(value BuyBitcoinProvider) RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinProvider](c, value)
}
func (FfiConverterTypeBuyBitcoinProvider) Read(reader io.Reader) BuyBitcoinProvider {
	id := readInt32(reader)
	return BuyBitcoinProvider(id)
}

func (FfiConverterTypeBuyBitcoinProvider) Write(writer io.Writer, value BuyBitcoinProvider) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeBuyBitcoinProvider struct{}

func (_ FfiDestroyerTypeBuyBitcoinProvider) Destroy(value BuyBitcoinProvider) {
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

type FfiConverterTypeGetPaymentRequest struct{}

var FfiConverterTypeGetPaymentRequestINSTANCE = FfiConverterTypeGetPaymentRequest{}

func (c FfiConverterTypeGetPaymentRequest) Lift(rb RustBufferI) GetPaymentRequest {
	return LiftFromRustBuffer[GetPaymentRequest](c, rb)
}

func (c FfiConverterTypeGetPaymentRequest) Lower(value GetPaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[GetPaymentRequest](c, value)
}
func (FfiConverterTypeGetPaymentRequest) Read(reader io.Reader) GetPaymentRequest {
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
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeGetPaymentRequest.Read()", id))
	}
}

func (FfiConverterTypeGetPaymentRequest) Write(writer io.Writer, value GetPaymentRequest) {
	switch variant_value := value.(type) {
	case GetPaymentRequestPaymentHash:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variant_value.PaymentHash)
	case GetPaymentRequestSwapId:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.SwapId)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeGetPaymentRequest.Write", value))
	}
}

type FfiDestroyerTypeGetPaymentRequest struct{}

func (_ FfiDestroyerTypeGetPaymentRequest) Destroy(value GetPaymentRequest) {
	value.Destroy()
}

type InputType interface {
	Destroy()
}
type InputTypeBitcoinAddress struct {
	Address BitcoinAddressData
}

func (e InputTypeBitcoinAddress) Destroy() {
	FfiDestroyerTypeBitcoinAddressData{}.Destroy(e.Address)
}

type InputTypeLiquidAddress struct {
	Address LiquidAddressData
}

func (e InputTypeLiquidAddress) Destroy() {
	FfiDestroyerTypeLiquidAddressData{}.Destroy(e.Address)
}

type InputTypeBolt11 struct {
	Invoice LnInvoice
}

func (e InputTypeBolt11) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(e.Invoice)
}

type InputTypeBolt12Offer struct {
	Offer LnOffer
}

func (e InputTypeBolt12Offer) Destroy() {
	FfiDestroyerTypeLnOffer{}.Destroy(e.Offer)
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
	Data LnUrlPayRequestData
}

func (e InputTypeLnUrlPay) Destroy() {
	FfiDestroyerTypeLnUrlPayRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlWithdraw struct {
	Data LnUrlWithdrawRequestData
}

func (e InputTypeLnUrlWithdraw) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlAuth struct {
	Data LnUrlAuthRequestData
}

func (e InputTypeLnUrlAuth) Destroy() {
	FfiDestroyerTypeLnUrlAuthRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlError struct {
	Data LnUrlErrorData
}

func (e InputTypeLnUrlError) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeInputType struct{}

var FfiConverterTypeInputTypeINSTANCE = FfiConverterTypeInputType{}

func (c FfiConverterTypeInputType) Lift(rb RustBufferI) InputType {
	return LiftFromRustBuffer[InputType](c, rb)
}

func (c FfiConverterTypeInputType) Lower(value InputType) RustBuffer {
	return LowerIntoRustBuffer[InputType](c, value)
}
func (FfiConverterTypeInputType) Read(reader io.Reader) InputType {
	id := readInt32(reader)
	switch id {
	case 1:
		return InputTypeBitcoinAddress{
			FfiConverterTypeBitcoinAddressDataINSTANCE.Read(reader),
		}
	case 2:
		return InputTypeLiquidAddress{
			FfiConverterTypeLiquidAddressDataINSTANCE.Read(reader),
		}
	case 3:
		return InputTypeBolt11{
			FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
		}
	case 4:
		return InputTypeBolt12Offer{
			FfiConverterTypeLNOfferINSTANCE.Read(reader),
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
			FfiConverterTypeLnUrlPayRequestDataINSTANCE.Read(reader),
		}
	case 8:
		return InputTypeLnUrlWithdraw{
			FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		}
	case 9:
		return InputTypeLnUrlAuth{
			FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Read(reader),
		}
	case 10:
		return InputTypeLnUrlError{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeInputType.Read()", id))
	}
}

func (FfiConverterTypeInputType) Write(writer io.Writer, value InputType) {
	switch variant_value := value.(type) {
	case InputTypeBitcoinAddress:
		writeInt32(writer, 1)
		FfiConverterTypeBitcoinAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeLiquidAddress:
		writeInt32(writer, 2)
		FfiConverterTypeLiquidAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeBolt11:
		writeInt32(writer, 3)
		FfiConverterTypeLNInvoiceINSTANCE.Write(writer, variant_value.Invoice)
	case InputTypeBolt12Offer:
		writeInt32(writer, 4)
		FfiConverterTypeLNOfferINSTANCE.Write(writer, variant_value.Offer)
	case InputTypeNodeId:
		writeInt32(writer, 5)
		FfiConverterStringINSTANCE.Write(writer, variant_value.NodeId)
	case InputTypeUrl:
		writeInt32(writer, 6)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
	case InputTypeLnUrlPay:
		writeInt32(writer, 7)
		FfiConverterTypeLnUrlPayRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlWithdraw:
		writeInt32(writer, 8)
		FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlAuth:
		writeInt32(writer, 9)
		FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlError:
		writeInt32(writer, 10)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeInputType.Write", value))
	}
}

type FfiDestroyerTypeInputType struct{}

func (_ FfiDestroyerTypeInputType) Destroy(value InputType) {
	value.Destroy()
}

type LiquidNetwork uint

const (
	LiquidNetworkMainnet LiquidNetwork = 1
	LiquidNetworkTestnet LiquidNetwork = 2
)

type FfiConverterTypeLiquidNetwork struct{}

var FfiConverterTypeLiquidNetworkINSTANCE = FfiConverterTypeLiquidNetwork{}

func (c FfiConverterTypeLiquidNetwork) Lift(rb RustBufferI) LiquidNetwork {
	return LiftFromRustBuffer[LiquidNetwork](c, rb)
}

func (c FfiConverterTypeLiquidNetwork) Lower(value LiquidNetwork) RustBuffer {
	return LowerIntoRustBuffer[LiquidNetwork](c, value)
}
func (FfiConverterTypeLiquidNetwork) Read(reader io.Reader) LiquidNetwork {
	id := readInt32(reader)
	return LiquidNetwork(id)
}

func (FfiConverterTypeLiquidNetwork) Write(writer io.Writer, value LiquidNetwork) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeLiquidNetwork struct{}

func (_ FfiDestroyerTypeLiquidNetwork) Destroy(value LiquidNetwork) {
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

type FfiConverterTypeListPaymentDetails struct{}

var FfiConverterTypeListPaymentDetailsINSTANCE = FfiConverterTypeListPaymentDetails{}

func (c FfiConverterTypeListPaymentDetails) Lift(rb RustBufferI) ListPaymentDetails {
	return LiftFromRustBuffer[ListPaymentDetails](c, rb)
}

func (c FfiConverterTypeListPaymentDetails) Lower(value ListPaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentDetails](c, value)
}
func (FfiConverterTypeListPaymentDetails) Read(reader io.Reader) ListPaymentDetails {
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
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeListPaymentDetails.Read()", id))
	}
}

func (FfiConverterTypeListPaymentDetails) Write(writer io.Writer, value ListPaymentDetails) {
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
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeListPaymentDetails.Write", value))
	}
}

type FfiDestroyerTypeListPaymentDetails struct{}

func (_ FfiDestroyerTypeListPaymentDetails) Destroy(value ListPaymentDetails) {
	value.Destroy()
}

type LnUrlAuthError struct {
	err error
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorGeneric{
			Err: err,
		},
	}
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorInvalidUri{
			Err: err,
		},
	}
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorServiceConnectivity{
			Err: err,
		},
	}
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

type FfiConverterTypeLnUrlAuthError struct{}

var FfiConverterTypeLnUrlAuthErrorINSTANCE = FfiConverterTypeLnUrlAuthError{}

func (c FfiConverterTypeLnUrlAuthError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlAuthError) Lower(value *LnUrlAuthError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlAuthError](c, value)
}

func (c FfiConverterTypeLnUrlAuthError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlAuthError.Read()", errorID))
	}
}

func (c FfiConverterTypeLnUrlAuthError) Write(writer io.Writer, value *LnUrlAuthError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlAuthError.Write", value))
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
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlCallbackStatus struct{}

var FfiConverterTypeLnUrlCallbackStatusINSTANCE = FfiConverterTypeLnUrlCallbackStatus{}

func (c FfiConverterTypeLnUrlCallbackStatus) Lift(rb RustBufferI) LnUrlCallbackStatus {
	return LiftFromRustBuffer[LnUrlCallbackStatus](c, rb)
}

func (c FfiConverterTypeLnUrlCallbackStatus) Lower(value LnUrlCallbackStatus) RustBuffer {
	return LowerIntoRustBuffer[LnUrlCallbackStatus](c, value)
}
func (FfiConverterTypeLnUrlCallbackStatus) Read(reader io.Reader) LnUrlCallbackStatus {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlCallbackStatusOk{}
	case 2:
		return LnUrlCallbackStatusErrorStatus{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlCallbackStatus.Read()", id))
	}
}

func (FfiConverterTypeLnUrlCallbackStatus) Write(writer io.Writer, value LnUrlCallbackStatus) {
	switch variant_value := value.(type) {
	case LnUrlCallbackStatusOk:
		writeInt32(writer, 1)
	case LnUrlCallbackStatusErrorStatus:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlCallbackStatus.Write", value))
	}
}

type FfiDestroyerTypeLnUrlCallbackStatus struct{}

func (_ FfiDestroyerTypeLnUrlCallbackStatus) Destroy(value LnUrlCallbackStatus) {
	value.Destroy()
}

type LnUrlPayError struct {
	err error
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorAlreadyPaid{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorGeneric{
			Err: err,
		},
	}
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

type LnUrlPayErrorInvalidAmount struct {
	Err string
}

func NewLnUrlPayErrorInvalidAmount(
	err string,
) *LnUrlPayError {
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidAmount{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidInvoice{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidNetwork{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidUri{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvoiceExpired{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorPaymentFailed{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorPaymentTimeout{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorRouteNotFound{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorRouteTooExpensive{
			Err: err,
		},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorServiceConnectivity{
			Err: err,
		},
	}
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

type FfiConverterTypeLnUrlPayError struct{}

var FfiConverterTypeLnUrlPayErrorINSTANCE = FfiConverterTypeLnUrlPayError{}

func (c FfiConverterTypeLnUrlPayError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlPayError) Lower(value *LnUrlPayError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlPayError](c, value)
}

func (c FfiConverterTypeLnUrlPayError) Read(reader io.Reader) error {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &LnUrlPayError{&LnUrlPayErrorAlreadyPaid{}}
	case 2:
		return &LnUrlPayError{&LnUrlPayErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 3:
		return &LnUrlPayError{&LnUrlPayErrorInvalidAmount{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 4:
		return &LnUrlPayError{&LnUrlPayErrorInvalidInvoice{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 5:
		return &LnUrlPayError{&LnUrlPayErrorInvalidNetwork{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 6:
		return &LnUrlPayError{&LnUrlPayErrorInvalidUri{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 7:
		return &LnUrlPayError{&LnUrlPayErrorInvoiceExpired{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 8:
		return &LnUrlPayError{&LnUrlPayErrorPaymentFailed{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 9:
		return &LnUrlPayError{&LnUrlPayErrorPaymentTimeout{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 10:
		return &LnUrlPayError{&LnUrlPayErrorRouteNotFound{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 11:
		return &LnUrlPayError{&LnUrlPayErrorRouteTooExpensive{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	case 12:
		return &LnUrlPayError{&LnUrlPayErrorServiceConnectivity{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlPayError.Read()", errorID))
	}
}

func (c FfiConverterTypeLnUrlPayError) Write(writer io.Writer, value *LnUrlPayError) {
	switch variantValue := value.err.(type) {
	case *LnUrlPayErrorAlreadyPaid:
		writeInt32(writer, 1)
	case *LnUrlPayErrorGeneric:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidAmount:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidInvoice:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidNetwork:
		writeInt32(writer, 5)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvalidUri:
		writeInt32(writer, 6)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorInvoiceExpired:
		writeInt32(writer, 7)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorPaymentFailed:
		writeInt32(writer, 8)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorPaymentTimeout:
		writeInt32(writer, 9)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorRouteNotFound:
		writeInt32(writer, 10)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorRouteTooExpensive:
		writeInt32(writer, 11)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	case *LnUrlPayErrorServiceConnectivity:
		writeInt32(writer, 12)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlPayError.Write", value))
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
	FfiDestroyerTypeLnUrlPaySuccessData{}.Destroy(e.Data)
}

type LnUrlPayResultEndpointError struct {
	Data LnUrlErrorData
}

func (e LnUrlPayResultEndpointError) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type LnUrlPayResultPayError struct {
	Data LnUrlPayErrorData
}

func (e LnUrlPayResultPayError) Destroy() {
	FfiDestroyerTypeLnUrlPayErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlPayResult struct{}

var FfiConverterTypeLnUrlPayResultINSTANCE = FfiConverterTypeLnUrlPayResult{}

func (c FfiConverterTypeLnUrlPayResult) Lift(rb RustBufferI) LnUrlPayResult {
	return LiftFromRustBuffer[LnUrlPayResult](c, rb)
}

func (c FfiConverterTypeLnUrlPayResult) Lower(value LnUrlPayResult) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayResult](c, value)
}
func (FfiConverterTypeLnUrlPayResult) Read(reader io.Reader) LnUrlPayResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlPayResultEndpointSuccess{
			FfiConverterTypeLnUrlPaySuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlPayResultEndpointError{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlPayResultPayError{
			FfiConverterTypeLnUrlPayErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlPayResult.Read()", id))
	}
}

func (FfiConverterTypeLnUrlPayResult) Write(writer io.Writer, value LnUrlPayResult) {
	switch variant_value := value.(type) {
	case LnUrlPayResultEndpointSuccess:
		writeInt32(writer, 1)
		FfiConverterTypeLnUrlPaySuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultEndpointError:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultPayError:
		writeInt32(writer, 3)
		FfiConverterTypeLnUrlPayErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlPayResult.Write", value))
	}
}

type FfiDestroyerTypeLnUrlPayResult struct{}

func (_ FfiDestroyerTypeLnUrlPayResult) Destroy(value LnUrlPayResult) {
	value.Destroy()
}

type LnUrlWithdrawError struct {
	err error
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorGeneric{
			Err: err,
		},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidAmount{
			Err: err,
		},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidInvoice{
			Err: err,
		},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidUri{
			Err: err,
		},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorServiceConnectivity{
			Err: err,
		},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvoiceNoRoutingHints{
			Err: err,
		},
	}
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

type FfiConverterTypeLnUrlWithdrawError struct{}

var FfiConverterTypeLnUrlWithdrawErrorINSTANCE = FfiConverterTypeLnUrlWithdrawError{}

func (c FfiConverterTypeLnUrlWithdrawError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlWithdrawError) Lower(value *LnUrlWithdrawError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlWithdrawError](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlWithdrawError.Read()", errorID))
	}
}

func (c FfiConverterTypeLnUrlWithdrawError) Write(writer io.Writer, value *LnUrlWithdrawError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlWithdrawError.Write", value))
	}
}

type LnUrlWithdrawResult interface {
	Destroy()
}
type LnUrlWithdrawResultOk struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultOk) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultTimeout struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultTimeout) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlWithdrawResultErrorStatus) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlWithdrawResult struct{}

var FfiConverterTypeLnUrlWithdrawResultINSTANCE = FfiConverterTypeLnUrlWithdrawResult{}

func (c FfiConverterTypeLnUrlWithdrawResult) Lift(rb RustBufferI) LnUrlWithdrawResult {
	return LiftFromRustBuffer[LnUrlWithdrawResult](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawResult) Lower(value LnUrlWithdrawResult) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawResult](c, value)
}
func (FfiConverterTypeLnUrlWithdrawResult) Read(reader io.Reader) LnUrlWithdrawResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlWithdrawResultOk{
			FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlWithdrawResultTimeout{
			FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlWithdrawResultErrorStatus{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlWithdrawResult.Read()", id))
	}
}

func (FfiConverterTypeLnUrlWithdrawResult) Write(writer io.Writer, value LnUrlWithdrawResult) {
	switch variant_value := value.(type) {
	case LnUrlWithdrawResultOk:
		writeInt32(writer, 1)
		FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultTimeout:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultErrorStatus:
		writeInt32(writer, 3)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlWithdrawResult.Write", value))
	}
}

type FfiDestroyerTypeLnUrlWithdrawResult struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawResult) Destroy(value LnUrlWithdrawResult) {
	value.Destroy()
}

type Network uint

const (
	NetworkBitcoin Network = 1
	NetworkTestnet Network = 2
	NetworkSignet  Network = 3
	NetworkRegtest Network = 4
)

type FfiConverterTypeNetwork struct{}

var FfiConverterTypeNetworkINSTANCE = FfiConverterTypeNetwork{}

func (c FfiConverterTypeNetwork) Lift(rb RustBufferI) Network {
	return LiftFromRustBuffer[Network](c, rb)
}

func (c FfiConverterTypeNetwork) Lower(value Network) RustBuffer {
	return LowerIntoRustBuffer[Network](c, value)
}
func (FfiConverterTypeNetwork) Read(reader io.Reader) Network {
	id := readInt32(reader)
	return Network(id)
}

func (FfiConverterTypeNetwork) Write(writer io.Writer, value Network) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeNetwork struct{}

func (_ FfiDestroyerTypeNetwork) Destroy(value Network) {
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
	AssetId        string
	ReceiverAmount float64
}

func (e PayAmountAsset) Destroy() {
	FfiDestroyerString{}.Destroy(e.AssetId)
	FfiDestroyerFloat64{}.Destroy(e.ReceiverAmount)
}

type PayAmountDrain struct {
}

func (e PayAmountDrain) Destroy() {
}

type FfiConverterTypePayAmount struct{}

var FfiConverterTypePayAmountINSTANCE = FfiConverterTypePayAmount{}

func (c FfiConverterTypePayAmount) Lift(rb RustBufferI) PayAmount {
	return LiftFromRustBuffer[PayAmount](c, rb)
}

func (c FfiConverterTypePayAmount) Lower(value PayAmount) RustBuffer {
	return LowerIntoRustBuffer[PayAmount](c, value)
}
func (FfiConverterTypePayAmount) Read(reader io.Reader) PayAmount {
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
		}
	case 3:
		return PayAmountDrain{}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypePayAmount.Read()", id))
	}
}

func (FfiConverterTypePayAmount) Write(writer io.Writer, value PayAmount) {
	switch variant_value := value.(type) {
	case PayAmountBitcoin:
		writeInt32(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.ReceiverAmountSat)
	case PayAmountAsset:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterFloat64INSTANCE.Write(writer, variant_value.ReceiverAmount)
	case PayAmountDrain:
		writeInt32(writer, 3)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypePayAmount.Write", value))
	}
}

type FfiDestroyerTypePayAmount struct{}

func (_ FfiDestroyerTypePayAmount) Destroy(value PayAmount) {
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
	FfiDestroyerOptionalTypeLnUrlInfo{}.Destroy(e.LnurlInfo)
	FfiDestroyerOptionalString{}.Destroy(e.ClaimTxId)
	FfiDestroyerOptionalString{}.Destroy(e.RefundTxId)
	FfiDestroyerOptionalUint64{}.Destroy(e.RefundTxAmountSat)
}

type PaymentDetailsLiquid struct {
	AssetId     string
	Destination string
	Description string
	AssetInfo   *AssetInfo
}

func (e PaymentDetailsLiquid) Destroy() {
	FfiDestroyerString{}.Destroy(e.AssetId)
	FfiDestroyerString{}.Destroy(e.Destination)
	FfiDestroyerString{}.Destroy(e.Description)
	FfiDestroyerOptionalTypeAssetInfo{}.Destroy(e.AssetInfo)
}

type PaymentDetailsBitcoin struct {
	SwapId                       string
	Description                  string
	AutoAcceptedFees             bool
	BitcoinExpirationBlockheight *uint32
	LiquidExpirationBlockheight  *uint32
	ClaimTxId                    *string
	RefundTxId                   *string
	RefundTxAmountSat            *uint64
}

func (e PaymentDetailsBitcoin) Destroy() {
	FfiDestroyerString{}.Destroy(e.SwapId)
	FfiDestroyerString{}.Destroy(e.Description)
	FfiDestroyerBool{}.Destroy(e.AutoAcceptedFees)
	FfiDestroyerOptionalUint32{}.Destroy(e.BitcoinExpirationBlockheight)
	FfiDestroyerOptionalUint32{}.Destroy(e.LiquidExpirationBlockheight)
	FfiDestroyerOptionalString{}.Destroy(e.ClaimTxId)
	FfiDestroyerOptionalString{}.Destroy(e.RefundTxId)
	FfiDestroyerOptionalUint64{}.Destroy(e.RefundTxAmountSat)
}

type FfiConverterTypePaymentDetails struct{}

var FfiConverterTypePaymentDetailsINSTANCE = FfiConverterTypePaymentDetails{}

func (c FfiConverterTypePaymentDetails) Lift(rb RustBufferI) PaymentDetails {
	return LiftFromRustBuffer[PaymentDetails](c, rb)
}

func (c FfiConverterTypePaymentDetails) Lower(value PaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[PaymentDetails](c, value)
}
func (FfiConverterTypePaymentDetails) Read(reader io.Reader) PaymentDetails {
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
			FfiConverterOptionalTypeLnUrlInfoINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalUint64INSTANCE.Read(reader),
		}
	case 2:
		return PaymentDetailsLiquid{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterOptionalTypeAssetInfoINSTANCE.Read(reader),
		}
	case 3:
		return PaymentDetailsBitcoin{
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterStringINSTANCE.Read(reader),
			FfiConverterBoolINSTANCE.Read(reader),
			FfiConverterOptionalUint32INSTANCE.Read(reader),
			FfiConverterOptionalUint32INSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
			FfiConverterOptionalUint64INSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypePaymentDetails.Read()", id))
	}
}

func (FfiConverterTypePaymentDetails) Write(writer io.Writer, value PaymentDetails) {
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
		FfiConverterOptionalTypeLnUrlInfoINSTANCE.Write(writer, variant_value.LnurlInfo)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.ClaimTxId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.RefundTxId)
		FfiConverterOptionalUint64INSTANCE.Write(writer, variant_value.RefundTxAmountSat)
	case PaymentDetailsLiquid:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.AssetId)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Destination)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Description)
		FfiConverterOptionalTypeAssetInfoINSTANCE.Write(writer, variant_value.AssetInfo)
	case PaymentDetailsBitcoin:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variant_value.SwapId)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Description)
		FfiConverterBoolINSTANCE.Write(writer, variant_value.AutoAcceptedFees)
		FfiConverterOptionalUint32INSTANCE.Write(writer, variant_value.BitcoinExpirationBlockheight)
		FfiConverterOptionalUint32INSTANCE.Write(writer, variant_value.LiquidExpirationBlockheight)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.ClaimTxId)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.RefundTxId)
		FfiConverterOptionalUint64INSTANCE.Write(writer, variant_value.RefundTxAmountSat)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypePaymentDetails.Write", value))
	}
}

type FfiDestroyerTypePaymentDetails struct{}

func (_ FfiDestroyerTypePaymentDetails) Destroy(value PaymentDetails) {
	value.Destroy()
}

type PaymentError struct {
	err error
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
var ErrPaymentErrorLwkError = fmt.Errorf("PaymentErrorLwkError")
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
	return &PaymentError{
		err: &PaymentErrorAlreadyClaimed{},
	}
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
	return &PaymentError{
		err: &PaymentErrorAlreadyPaid{},
	}
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
	return &PaymentError{
		err: &PaymentErrorPaymentInProgress{},
	}
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
	return &PaymentError{
		err: &PaymentErrorAmountOutOfRange{},
	}
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
	return &PaymentError{
		err: &PaymentErrorAmountMissing{},
	}
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
	return &PaymentError{
		err: &PaymentErrorAssetError{},
	}
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
	return &PaymentError{
		err: &PaymentErrorGeneric{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInvalidOrExpiredFees{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInsufficientFunds{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInvalidDescription{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInvalidInvoice{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInvalidNetwork{},
	}
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
	return &PaymentError{
		err: &PaymentErrorInvalidPreimage{},
	}
}

func (err PaymentErrorInvalidPreimage) Error() string {
	return fmt.Sprintf("InvalidPreimage: %s", err.message)
}

func (self PaymentErrorInvalidPreimage) Is(target error) bool {
	return target == ErrPaymentErrorInvalidPreimage
}

type PaymentErrorLwkError struct {
	message string
}

func NewPaymentErrorLwkError() *PaymentError {
	return &PaymentError{
		err: &PaymentErrorLwkError{},
	}
}

func (err PaymentErrorLwkError) Error() string {
	return fmt.Sprintf("LwkError: %s", err.message)
}

func (self PaymentErrorLwkError) Is(target error) bool {
	return target == ErrPaymentErrorLwkError
}

type PaymentErrorPairsNotFound struct {
	message string
}

func NewPaymentErrorPairsNotFound() *PaymentError {
	return &PaymentError{
		err: &PaymentErrorPairsNotFound{},
	}
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
	return &PaymentError{
		err: &PaymentErrorPaymentTimeout{},
	}
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
	return &PaymentError{
		err: &PaymentErrorPersistError{},
	}
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
	return &PaymentError{
		err: &PaymentErrorReceiveError{},
	}
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
	return &PaymentError{
		err: &PaymentErrorRefunded{},
	}
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
	return &PaymentError{
		err: &PaymentErrorSelfTransferNotSupported{},
	}
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
	return &PaymentError{
		err: &PaymentErrorSendError{},
	}
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
	return &PaymentError{
		err: &PaymentErrorSignerError{},
	}
}

func (err PaymentErrorSignerError) Error() string {
	return fmt.Sprintf("SignerError: %s", err.message)
}

func (self PaymentErrorSignerError) Is(target error) bool {
	return target == ErrPaymentErrorSignerError
}

type FfiConverterTypePaymentError struct{}

var FfiConverterTypePaymentErrorINSTANCE = FfiConverterTypePaymentError{}

func (c FfiConverterTypePaymentError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypePaymentError) Lower(value *PaymentError) RustBuffer {
	return LowerIntoRustBuffer[*PaymentError](c, value)
}

func (c FfiConverterTypePaymentError) Read(reader io.Reader) error {
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
		return &PaymentError{&PaymentErrorLwkError{message}}
	case 15:
		return &PaymentError{&PaymentErrorPairsNotFound{message}}
	case 16:
		return &PaymentError{&PaymentErrorPaymentTimeout{message}}
	case 17:
		return &PaymentError{&PaymentErrorPersistError{message}}
	case 18:
		return &PaymentError{&PaymentErrorReceiveError{message}}
	case 19:
		return &PaymentError{&PaymentErrorRefunded{message}}
	case 20:
		return &PaymentError{&PaymentErrorSelfTransferNotSupported{message}}
	case 21:
		return &PaymentError{&PaymentErrorSendError{message}}
	case 22:
		return &PaymentError{&PaymentErrorSignerError{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypePaymentError.Read()", errorID))
	}

}

func (c FfiConverterTypePaymentError) Write(writer io.Writer, value *PaymentError) {
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
	case *PaymentErrorLwkError:
		writeInt32(writer, 14)
	case *PaymentErrorPairsNotFound:
		writeInt32(writer, 15)
	case *PaymentErrorPaymentTimeout:
		writeInt32(writer, 16)
	case *PaymentErrorPersistError:
		writeInt32(writer, 17)
	case *PaymentErrorReceiveError:
		writeInt32(writer, 18)
	case *PaymentErrorRefunded:
		writeInt32(writer, 19)
	case *PaymentErrorSelfTransferNotSupported:
		writeInt32(writer, 20)
	case *PaymentErrorSendError:
		writeInt32(writer, 21)
	case *PaymentErrorSignerError:
		writeInt32(writer, 22)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypePaymentError.Write", value))
	}
}

type PaymentMethod uint

const (
	PaymentMethodLightning      PaymentMethod = 1
	PaymentMethodBitcoinAddress PaymentMethod = 2
	PaymentMethodLiquidAddress  PaymentMethod = 3
)

type FfiConverterTypePaymentMethod struct{}

var FfiConverterTypePaymentMethodINSTANCE = FfiConverterTypePaymentMethod{}

func (c FfiConverterTypePaymentMethod) Lift(rb RustBufferI) PaymentMethod {
	return LiftFromRustBuffer[PaymentMethod](c, rb)
}

func (c FfiConverterTypePaymentMethod) Lower(value PaymentMethod) RustBuffer {
	return LowerIntoRustBuffer[PaymentMethod](c, value)
}
func (FfiConverterTypePaymentMethod) Read(reader io.Reader) PaymentMethod {
	id := readInt32(reader)
	return PaymentMethod(id)
}

func (FfiConverterTypePaymentMethod) Write(writer io.Writer, value PaymentMethod) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentMethod struct{}

func (_ FfiDestroyerTypePaymentMethod) Destroy(value PaymentMethod) {
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

type FfiConverterTypePaymentState struct{}

var FfiConverterTypePaymentStateINSTANCE = FfiConverterTypePaymentState{}

func (c FfiConverterTypePaymentState) Lift(rb RustBufferI) PaymentState {
	return LiftFromRustBuffer[PaymentState](c, rb)
}

func (c FfiConverterTypePaymentState) Lower(value PaymentState) RustBuffer {
	return LowerIntoRustBuffer[PaymentState](c, value)
}
func (FfiConverterTypePaymentState) Read(reader io.Reader) PaymentState {
	id := readInt32(reader)
	return PaymentState(id)
}

func (FfiConverterTypePaymentState) Write(writer io.Writer, value PaymentState) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentState struct{}

func (_ FfiDestroyerTypePaymentState) Destroy(value PaymentState) {
}

type PaymentType uint

const (
	PaymentTypeReceive PaymentType = 1
	PaymentTypeSend    PaymentType = 2
)

type FfiConverterTypePaymentType struct{}

var FfiConverterTypePaymentTypeINSTANCE = FfiConverterTypePaymentType{}

func (c FfiConverterTypePaymentType) Lift(rb RustBufferI) PaymentType {
	return LiftFromRustBuffer[PaymentType](c, rb)
}

func (c FfiConverterTypePaymentType) Lower(value PaymentType) RustBuffer {
	return LowerIntoRustBuffer[PaymentType](c, value)
}
func (FfiConverterTypePaymentType) Read(reader io.Reader) PaymentType {
	id := readInt32(reader)
	return PaymentType(id)
}

func (FfiConverterTypePaymentType) Write(writer io.Writer, value PaymentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentType struct{}

func (_ FfiDestroyerTypePaymentType) Destroy(value PaymentType) {
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

type FfiConverterTypeReceiveAmount struct{}

var FfiConverterTypeReceiveAmountINSTANCE = FfiConverterTypeReceiveAmount{}

func (c FfiConverterTypeReceiveAmount) Lift(rb RustBufferI) ReceiveAmount {
	return LiftFromRustBuffer[ReceiveAmount](c, rb)
}

func (c FfiConverterTypeReceiveAmount) Lower(value ReceiveAmount) RustBuffer {
	return LowerIntoRustBuffer[ReceiveAmount](c, value)
}
func (FfiConverterTypeReceiveAmount) Read(reader io.Reader) ReceiveAmount {
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
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeReceiveAmount.Read()", id))
	}
}

func (FfiConverterTypeReceiveAmount) Write(writer io.Writer, value ReceiveAmount) {
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
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeReceiveAmount.Write", value))
	}
}

type FfiDestroyerTypeReceiveAmount struct{}

func (_ FfiDestroyerTypeReceiveAmount) Destroy(value ReceiveAmount) {
	value.Destroy()
}

// /////////////////////////////
type SdkError struct {
	err error
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
	return &SdkError{
		err: &SdkErrorAlreadyStarted{},
	}
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
	return &SdkError{
		err: &SdkErrorGeneric{},
	}
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
	return &SdkError{
		err: &SdkErrorNotStarted{},
	}
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
	return &SdkError{
		err: &SdkErrorServiceConnectivity{},
	}
}

func (err SdkErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SdkErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSdkErrorServiceConnectivity
}

type FfiConverterTypeSdkError struct{}

var FfiConverterTypeSdkErrorINSTANCE = FfiConverterTypeSdkError{}

func (c FfiConverterTypeSdkError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSdkError) Lower(value *SdkError) RustBuffer {
	return LowerIntoRustBuffer[*SdkError](c, value)
}

func (c FfiConverterTypeSdkError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSdkError.Read()", errorID))
	}

}

func (c FfiConverterTypeSdkError) Write(writer io.Writer, value *SdkError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSdkError.Write", value))
	}
}

type SdkEvent interface {
	Destroy()
}
type SdkEventPaymentFailed struct {
	Details Payment
}

func (e SdkEventPaymentFailed) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentPending struct {
	Details Payment
}

func (e SdkEventPaymentPending) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefundable struct {
	Details Payment
}

func (e SdkEventPaymentRefundable) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefunded struct {
	Details Payment
}

func (e SdkEventPaymentRefunded) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentRefundPending struct {
	Details Payment
}

func (e SdkEventPaymentRefundPending) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentSucceeded struct {
	Details Payment
}

func (e SdkEventPaymentSucceeded) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentWaitingConfirmation struct {
	Details Payment
}

func (e SdkEventPaymentWaitingConfirmation) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventPaymentWaitingFeeAcceptance struct {
	Details Payment
}

func (e SdkEventPaymentWaitingFeeAcceptance) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type SdkEventSynced struct {
}

func (e SdkEventSynced) Destroy() {
}

type FfiConverterTypeSdkEvent struct{}

var FfiConverterTypeSdkEventINSTANCE = FfiConverterTypeSdkEvent{}

func (c FfiConverterTypeSdkEvent) Lift(rb RustBufferI) SdkEvent {
	return LiftFromRustBuffer[SdkEvent](c, rb)
}

func (c FfiConverterTypeSdkEvent) Lower(value SdkEvent) RustBuffer {
	return LowerIntoRustBuffer[SdkEvent](c, value)
}
func (FfiConverterTypeSdkEvent) Read(reader io.Reader) SdkEvent {
	id := readInt32(reader)
	switch id {
	case 1:
		return SdkEventPaymentFailed{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 2:
		return SdkEventPaymentPending{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 3:
		return SdkEventPaymentRefundable{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 4:
		return SdkEventPaymentRefunded{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 5:
		return SdkEventPaymentRefundPending{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 6:
		return SdkEventPaymentSucceeded{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 7:
		return SdkEventPaymentWaitingConfirmation{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 8:
		return SdkEventPaymentWaitingFeeAcceptance{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 9:
		return SdkEventSynced{}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeSdkEvent.Read()", id))
	}
}

func (FfiConverterTypeSdkEvent) Write(writer io.Writer, value SdkEvent) {
	switch variant_value := value.(type) {
	case SdkEventPaymentFailed:
		writeInt32(writer, 1)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentPending:
		writeInt32(writer, 2)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefundable:
		writeInt32(writer, 3)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefunded:
		writeInt32(writer, 4)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentRefundPending:
		writeInt32(writer, 5)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentSucceeded:
		writeInt32(writer, 6)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentWaitingConfirmation:
		writeInt32(writer, 7)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventPaymentWaitingFeeAcceptance:
		writeInt32(writer, 8)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case SdkEventSynced:
		writeInt32(writer, 9)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeSdkEvent.Write", value))
	}
}

type FfiDestroyerTypeSdkEvent struct{}

func (_ FfiDestroyerTypeSdkEvent) Destroy(value SdkEvent) {
	value.Destroy()
}

type SendDestination interface {
	Destroy()
}
type SendDestinationLiquidAddress struct {
	AddressData LiquidAddressData
}

func (e SendDestinationLiquidAddress) Destroy() {
	FfiDestroyerTypeLiquidAddressData{}.Destroy(e.AddressData)
}

type SendDestinationBolt11 struct {
	Invoice LnInvoice
}

func (e SendDestinationBolt11) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(e.Invoice)
}

type SendDestinationBolt12 struct {
	Offer             LnOffer
	ReceiverAmountSat uint64
}

func (e SendDestinationBolt12) Destroy() {
	FfiDestroyerTypeLnOffer{}.Destroy(e.Offer)
	FfiDestroyerUint64{}.Destroy(e.ReceiverAmountSat)
}

type FfiConverterTypeSendDestination struct{}

var FfiConverterTypeSendDestinationINSTANCE = FfiConverterTypeSendDestination{}

func (c FfiConverterTypeSendDestination) Lift(rb RustBufferI) SendDestination {
	return LiftFromRustBuffer[SendDestination](c, rb)
}

func (c FfiConverterTypeSendDestination) Lower(value SendDestination) RustBuffer {
	return LowerIntoRustBuffer[SendDestination](c, value)
}
func (FfiConverterTypeSendDestination) Read(reader io.Reader) SendDestination {
	id := readInt32(reader)
	switch id {
	case 1:
		return SendDestinationLiquidAddress{
			FfiConverterTypeLiquidAddressDataINSTANCE.Read(reader),
		}
	case 2:
		return SendDestinationBolt11{
			FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
		}
	case 3:
		return SendDestinationBolt12{
			FfiConverterTypeLNOfferINSTANCE.Read(reader),
			FfiConverterUint64INSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeSendDestination.Read()", id))
	}
}

func (FfiConverterTypeSendDestination) Write(writer io.Writer, value SendDestination) {
	switch variant_value := value.(type) {
	case SendDestinationLiquidAddress:
		writeInt32(writer, 1)
		FfiConverterTypeLiquidAddressDataINSTANCE.Write(writer, variant_value.AddressData)
	case SendDestinationBolt11:
		writeInt32(writer, 2)
		FfiConverterTypeLNInvoiceINSTANCE.Write(writer, variant_value.Invoice)
	case SendDestinationBolt12:
		writeInt32(writer, 3)
		FfiConverterTypeLNOfferINSTANCE.Write(writer, variant_value.Offer)
		FfiConverterUint64INSTANCE.Write(writer, variant_value.ReceiverAmountSat)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeSendDestination.Write", value))
	}
}

type FfiDestroyerTypeSendDestination struct{}

func (_ FfiDestroyerTypeSendDestination) Destroy(value SendDestination) {
	value.Destroy()
}

type SignerError struct {
	err error
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
	return &SignerError{
		err: &SignerErrorGeneric{
			Err: err,
		},
	}
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

type FfiConverterTypeSignerError struct{}

var FfiConverterTypeSignerErrorINSTANCE = FfiConverterTypeSignerError{}

func (c FfiConverterTypeSignerError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSignerError) Lower(value *SignerError) RustBuffer {
	return LowerIntoRustBuffer[*SignerError](c, value)
}

func (c FfiConverterTypeSignerError) Read(reader io.Reader) error {
	errorID := readUint32(reader)

	switch errorID {
	case 1:
		return &SignerError{&SignerErrorGeneric{
			Err: FfiConverterStringINSTANCE.Read(reader),
		}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSignerError.Read()", errorID))
	}
}

func (c FfiConverterTypeSignerError) Write(writer io.Writer, value *SignerError) {
	switch variantValue := value.err.(type) {
	case *SignerErrorGeneric:
		writeInt32(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, variantValue.Err)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSignerError.Write", value))
	}
}

type SuccessAction interface {
	Destroy()
}
type SuccessActionAes struct {
	Data AesSuccessActionData
}

func (e SuccessActionAes) Destroy() {
	FfiDestroyerTypeAesSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionMessage) Destroy() {
	FfiDestroyerTypeMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionUrl) Destroy() {
	FfiDestroyerTypeUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterTypeSuccessAction struct{}

var FfiConverterTypeSuccessActionINSTANCE = FfiConverterTypeSuccessAction{}

func (c FfiConverterTypeSuccessAction) Lift(rb RustBufferI) SuccessAction {
	return LiftFromRustBuffer[SuccessAction](c, rb)
}

func (c FfiConverterTypeSuccessAction) Lower(value SuccessAction) RustBuffer {
	return LowerIntoRustBuffer[SuccessAction](c, value)
}
func (FfiConverterTypeSuccessAction) Read(reader io.Reader) SuccessAction {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionAes{
			FfiConverterTypeAesSuccessActionDataINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionMessage{
			FfiConverterTypeMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionUrl{
			FfiConverterTypeUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeSuccessAction.Read()", id))
	}
}

func (FfiConverterTypeSuccessAction) Write(writer io.Writer, value SuccessAction) {
	switch variant_value := value.(type) {
	case SuccessActionAes:
		writeInt32(writer, 1)
		FfiConverterTypeAesSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionMessage:
		writeInt32(writer, 2)
		FfiConverterTypeMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionUrl:
		writeInt32(writer, 3)
		FfiConverterTypeUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeSuccessAction.Write", value))
	}
}

type FfiDestroyerTypeSuccessAction struct{}

func (_ FfiDestroyerTypeSuccessAction) Destroy(value SuccessAction) {
	value.Destroy()
}

type SuccessActionProcessed interface {
	Destroy()
}
type SuccessActionProcessedAes struct {
	Result AesSuccessActionDataResult
}

func (e SuccessActionProcessedAes) Destroy() {
	FfiDestroyerTypeAesSuccessActionDataResult{}.Destroy(e.Result)
}

type SuccessActionProcessedMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionProcessedMessage) Destroy() {
	FfiDestroyerTypeMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionProcessedUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionProcessedUrl) Destroy() {
	FfiDestroyerTypeUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterTypeSuccessActionProcessed struct{}

var FfiConverterTypeSuccessActionProcessedINSTANCE = FfiConverterTypeSuccessActionProcessed{}

func (c FfiConverterTypeSuccessActionProcessed) Lift(rb RustBufferI) SuccessActionProcessed {
	return LiftFromRustBuffer[SuccessActionProcessed](c, rb)
}

func (c FfiConverterTypeSuccessActionProcessed) Lower(value SuccessActionProcessed) RustBuffer {
	return LowerIntoRustBuffer[SuccessActionProcessed](c, value)
}
func (FfiConverterTypeSuccessActionProcessed) Read(reader io.Reader) SuccessActionProcessed {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionProcessedAes{
			FfiConverterTypeAesSuccessActionDataResultINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionProcessedMessage{
			FfiConverterTypeMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionProcessedUrl{
			FfiConverterTypeUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeSuccessActionProcessed.Read()", id))
	}
}

func (FfiConverterTypeSuccessActionProcessed) Write(writer io.Writer, value SuccessActionProcessed) {
	switch variant_value := value.(type) {
	case SuccessActionProcessedAes:
		writeInt32(writer, 1)
		FfiConverterTypeAesSuccessActionDataResultINSTANCE.Write(writer, variant_value.Result)
	case SuccessActionProcessedMessage:
		writeInt32(writer, 2)
		FfiConverterTypeMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionProcessedUrl:
		writeInt32(writer, 3)
		FfiConverterTypeUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeSuccessActionProcessed.Write", value))
	}
}

type FfiDestroyerTypeSuccessActionProcessed struct{}

func (_ FfiDestroyerTypeSuccessActionProcessed) Destroy(value SuccessActionProcessed) {
	value.Destroy()
}

type uniffiCallbackResult C.int32_t

const (
	uniffiIdxCallbackFree               uniffiCallbackResult = 0
	uniffiCallbackResultSuccess         uniffiCallbackResult = 0
	uniffiCallbackResultError           uniffiCallbackResult = 1
	uniffiCallbackUnexpectedResultError uniffiCallbackResult = 2
	uniffiCallbackCancelled             uniffiCallbackResult = 3
)

type concurrentHandleMap[T any] struct {
	leftMap       map[uint64]*T
	rightMap      map[*T]uint64
	currentHandle uint64
	lock          sync.RWMutex
}

func newConcurrentHandleMap[T any]() *concurrentHandleMap[T] {
	return &concurrentHandleMap[T]{
		leftMap:  map[uint64]*T{},
		rightMap: map[*T]uint64{},
	}
}

func (cm *concurrentHandleMap[T]) insert(obj *T) uint64 {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if existingHandle, ok := cm.rightMap[obj]; ok {
		return existingHandle
	}
	cm.currentHandle = cm.currentHandle + 1
	cm.leftMap[cm.currentHandle] = obj
	cm.rightMap[obj] = cm.currentHandle
	return cm.currentHandle
}

func (cm *concurrentHandleMap[T]) remove(handle uint64) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if val, ok := cm.leftMap[handle]; ok {
		delete(cm.leftMap, handle)
		delete(cm.rightMap, val)
	}
	return false
}

func (cm *concurrentHandleMap[T]) tryGet(handle uint64) (*T, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	val, ok := cm.leftMap[handle]
	return val, ok
}

type FfiConverterCallbackInterface[CallbackInterface any] struct {
	handleMap *concurrentHandleMap[CallbackInterface]
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) drop(handle uint64) RustBuffer {
	c.handleMap.remove(handle)
	return RustBuffer{}
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Lift(handle uint64) CallbackInterface {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return *val
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Read(reader io.Reader) CallbackInterface {
	return c.Lift(readUint64(reader))
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Lower(value CallbackInterface) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(&value))
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Write(writer io.Writer, value CallbackInterface) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type EventListener interface {
	OnEvent(e SdkEvent)
}

// foreignCallbackCallbackInterfaceEventListener cannot be callable be a compiled function at a same time
type foreignCallbackCallbackInterfaceEventListener struct{}

//export breez_sdk_liquid_bindings_cgo_EventListener
func breez_sdk_liquid_bindings_cgo_EventListener(handle C.uint64_t, method C.int32_t, argsPtr *C.uint8_t, argsLen C.int32_t, outBuf *C.RustBuffer) C.int32_t {
	cb := FfiConverterCallbackInterfaceEventListenerINSTANCE.Lift(uint64(handle))
	switch method {
	case 0:
		// 0 means Rust is done with the callback, and the callback
		// can be dropped by the foreign language.
		*outBuf = FfiConverterCallbackInterfaceEventListenerINSTANCE.drop(uint64(handle))
		// See docs of ForeignCallback in `uniffi/src/ffi/foreigncallbacks.rs`
		return C.int32_t(uniffiIdxCallbackFree)

	case 1:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceEventListener{}.InvokeOnEvent(cb, args, outBuf)
		return C.int32_t(result)

	default:
		// This should never happen, because an out of bounds method index won't
		// ever be used. Once we can catch errors, we should return an InternalException.
		// https://github.com/mozilla/uniffi-rs/issues/351
		return C.int32_t(uniffiCallbackUnexpectedResultError)
	}
}

func (foreignCallbackCallbackInterfaceEventListener) InvokeOnEvent(callback EventListener, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	callback.OnEvent(FfiConverterTypeSdkEventINSTANCE.Read(reader))

	return uniffiCallbackResultSuccess
}

type FfiConverterCallbackInterfaceEventListener struct {
	FfiConverterCallbackInterface[EventListener]
}

var FfiConverterCallbackInterfaceEventListenerINSTANCE = &FfiConverterCallbackInterfaceEventListener{
	FfiConverterCallbackInterface: FfiConverterCallbackInterface[EventListener]{
		handleMap: newConcurrentHandleMap[EventListener](),
	},
}

// This is a static function because only 1 instance is supported for registering
func (c *FfiConverterCallbackInterfaceEventListener) register() {
	rustCall(func(status *C.RustCallStatus) int32 {
		C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_eventlistener(C.ForeignCallback(C.breez_sdk_liquid_bindings_cgo_EventListener), status)
		return 0
	})
}

type FfiDestroyerCallbackInterfaceEventListener struct{}

func (FfiDestroyerCallbackInterfaceEventListener) Destroy(value EventListener) {
}

type Logger interface {
	Log(l LogEntry)
}

// foreignCallbackCallbackInterfaceLogger cannot be callable be a compiled function at a same time
type foreignCallbackCallbackInterfaceLogger struct{}

//export breez_sdk_liquid_bindings_cgo_Logger
func breez_sdk_liquid_bindings_cgo_Logger(handle C.uint64_t, method C.int32_t, argsPtr *C.uint8_t, argsLen C.int32_t, outBuf *C.RustBuffer) C.int32_t {
	cb := FfiConverterCallbackInterfaceLoggerINSTANCE.Lift(uint64(handle))
	switch method {
	case 0:
		// 0 means Rust is done with the callback, and the callback
		// can be dropped by the foreign language.
		*outBuf = FfiConverterCallbackInterfaceLoggerINSTANCE.drop(uint64(handle))
		// See docs of ForeignCallback in `uniffi/src/ffi/foreigncallbacks.rs`
		return C.int32_t(uniffiIdxCallbackFree)

	case 1:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceLogger{}.InvokeLog(cb, args, outBuf)
		return C.int32_t(result)

	default:
		// This should never happen, because an out of bounds method index won't
		// ever be used. Once we can catch errors, we should return an InternalException.
		// https://github.com/mozilla/uniffi-rs/issues/351
		return C.int32_t(uniffiCallbackUnexpectedResultError)
	}
}

func (foreignCallbackCallbackInterfaceLogger) InvokeLog(callback Logger, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	callback.Log(FfiConverterTypeLogEntryINSTANCE.Read(reader))

	return uniffiCallbackResultSuccess
}

type FfiConverterCallbackInterfaceLogger struct {
	FfiConverterCallbackInterface[Logger]
}

var FfiConverterCallbackInterfaceLoggerINSTANCE = &FfiConverterCallbackInterfaceLogger{
	FfiConverterCallbackInterface: FfiConverterCallbackInterface[Logger]{
		handleMap: newConcurrentHandleMap[Logger](),
	},
}

// This is a static function because only 1 instance is supported for registering
func (c *FfiConverterCallbackInterfaceLogger) register() {
	rustCall(func(status *C.RustCallStatus) int32 {
		C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_logger(C.ForeignCallback(C.breez_sdk_liquid_bindings_cgo_Logger), status)
		return 0
	})
}

type FfiDestroyerCallbackInterfaceLogger struct{}

func (FfiDestroyerCallbackInterfaceLogger) Destroy(value Logger) {
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

// foreignCallbackCallbackInterfaceSigner cannot be callable be a compiled function at a same time
type foreignCallbackCallbackInterfaceSigner struct{}

//export breez_sdk_liquid_bindings_cgo_Signer
func breez_sdk_liquid_bindings_cgo_Signer(handle C.uint64_t, method C.int32_t, argsPtr *C.uint8_t, argsLen C.int32_t, outBuf *C.RustBuffer) C.int32_t {
	cb := FfiConverterCallbackInterfaceSignerINSTANCE.Lift(uint64(handle))
	switch method {
	case 0:
		// 0 means Rust is done with the callback, and the callback
		// can be dropped by the foreign language.
		*outBuf = FfiConverterCallbackInterfaceSignerINSTANCE.drop(uint64(handle))
		// See docs of ForeignCallback in `uniffi/src/ffi/foreigncallbacks.rs`
		return C.int32_t(uniffiIdxCallbackFree)

	case 1:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeXpub(cb, args, outBuf)
		return C.int32_t(result)
	case 2:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeDeriveXpub(cb, args, outBuf)
		return C.int32_t(result)
	case 3:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeSignEcdsa(cb, args, outBuf)
		return C.int32_t(result)
	case 4:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeSignEcdsaRecoverable(cb, args, outBuf)
		return C.int32_t(result)
	case 5:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeSlip77MasterBlindingKey(cb, args, outBuf)
		return C.int32_t(result)
	case 6:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeHmacSha256(cb, args, outBuf)
		return C.int32_t(result)
	case 7:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeEciesEncrypt(cb, args, outBuf)
		return C.int32_t(result)
	case 8:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceSigner{}.InvokeEciesDecrypt(cb, args, outBuf)
		return C.int32_t(result)

	default:
		// This should never happen, because an out of bounds method index won't
		// ever be used. Once we can catch errors, we should return an InternalException.
		// https://github.com/mozilla/uniffi-rs/issues/351
		return C.int32_t(uniffiCallbackUnexpectedResultError)
	}
}

func (foreignCallbackCallbackInterfaceSigner) InvokeXpub(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	result, err := callback.Xpub()

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeDeriveXpub(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.DeriveXpub(FfiConverterStringINSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeSignEcdsa(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.SignEcdsa(FfiConverterSequenceUint8INSTANCE.Read(reader), FfiConverterStringINSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeSignEcdsaRecoverable(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.SignEcdsaRecoverable(FfiConverterSequenceUint8INSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeSlip77MasterBlindingKey(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	result, err := callback.Slip77MasterBlindingKey()

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeHmacSha256(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.HmacSha256(FfiConverterSequenceUint8INSTANCE.Read(reader), FfiConverterStringINSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeEciesEncrypt(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.EciesEncrypt(FfiConverterSequenceUint8INSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}
func (foreignCallbackCallbackInterfaceSigner) InvokeEciesDecrypt(callback Signer, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	result, err := callback.EciesDecrypt(FfiConverterSequenceUint8INSTANCE.Read(reader))

	if err != nil {
		// The only way to bypass an unexpected error is to bypass pointer to an empty
		// instance of the error
		if err.err == nil {
			return uniffiCallbackUnexpectedResultError
		}
		*outBuf = LowerIntoRustBuffer[*SignerError](FfiConverterTypeSignerErrorINSTANCE, err)
		return uniffiCallbackResultError
	}
	*outBuf = LowerIntoRustBuffer[[]uint8](FfiConverterSequenceUint8INSTANCE, result)
	return uniffiCallbackResultSuccess
}

type FfiConverterCallbackInterfaceSigner struct {
	FfiConverterCallbackInterface[Signer]
}

var FfiConverterCallbackInterfaceSignerINSTANCE = &FfiConverterCallbackInterfaceSigner{
	FfiConverterCallbackInterface: FfiConverterCallbackInterface[Signer]{
		handleMap: newConcurrentHandleMap[Signer](),
	},
}

// This is a static function because only 1 instance is supported for registering
func (c *FfiConverterCallbackInterfaceSigner) register() {
	rustCall(func(status *C.RustCallStatus) int32 {
		C.uniffi_breez_sdk_liquid_bindings_fn_init_callback_signer(C.ForeignCallback(C.breez_sdk_liquid_bindings_cgo_Signer), status)
		return 0
	})
}

type FfiDestroyerCallbackInterfaceSigner struct{}

func (FfiDestroyerCallbackInterfaceSigner) Destroy(value Signer) {
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

func (c FfiConverterOptionalUint32) Lower(value *uint32) RustBuffer {
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

func (c FfiConverterOptionalUint64) Lower(value *uint64) RustBuffer {
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

func (c FfiConverterOptionalInt64) Lower(value *int64) RustBuffer {
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

func (c FfiConverterOptionalFloat64) Lower(value *float64) RustBuffer {
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

func (c FfiConverterOptionalBool) Lower(value *bool) RustBuffer {
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

func (c FfiConverterOptionalString) Lower(value *string) RustBuffer {
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

type FfiConverterOptionalTypeAssetInfo struct{}

var FfiConverterOptionalTypeAssetInfoINSTANCE = FfiConverterOptionalTypeAssetInfo{}

func (c FfiConverterOptionalTypeAssetInfo) Lift(rb RustBufferI) *AssetInfo {
	return LiftFromRustBuffer[*AssetInfo](c, rb)
}

func (_ FfiConverterOptionalTypeAssetInfo) Read(reader io.Reader) *AssetInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeAssetInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeAssetInfo) Lower(value *AssetInfo) RustBuffer {
	return LowerIntoRustBuffer[*AssetInfo](c, value)
}

func (_ FfiConverterOptionalTypeAssetInfo) Write(writer io.Writer, value *AssetInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeAssetInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeAssetInfo struct{}

func (_ FfiDestroyerOptionalTypeAssetInfo) Destroy(value *AssetInfo) {
	if value != nil {
		FfiDestroyerTypeAssetInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeLnUrlInfo struct{}

var FfiConverterOptionalTypeLnUrlInfoINSTANCE = FfiConverterOptionalTypeLnUrlInfo{}

func (c FfiConverterOptionalTypeLnUrlInfo) Lift(rb RustBufferI) *LnUrlInfo {
	return LiftFromRustBuffer[*LnUrlInfo](c, rb)
}

func (_ FfiConverterOptionalTypeLnUrlInfo) Read(reader io.Reader) *LnUrlInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeLnUrlInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeLnUrlInfo) Lower(value *LnUrlInfo) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlInfo](c, value)
}

func (_ FfiConverterOptionalTypeLnUrlInfo) Write(writer io.Writer, value *LnUrlInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeLnUrlInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeLnUrlInfo struct{}

func (_ FfiDestroyerOptionalTypeLnUrlInfo) Destroy(value *LnUrlInfo) {
	if value != nil {
		FfiDestroyerTypeLnUrlInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypePayment struct{}

var FfiConverterOptionalTypePaymentINSTANCE = FfiConverterOptionalTypePayment{}

func (c FfiConverterOptionalTypePayment) Lift(rb RustBufferI) *Payment {
	return LiftFromRustBuffer[*Payment](c, rb)
}

func (_ FfiConverterOptionalTypePayment) Read(reader io.Reader) *Payment {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypePaymentINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypePayment) Lower(value *Payment) RustBuffer {
	return LowerIntoRustBuffer[*Payment](c, value)
}

func (_ FfiConverterOptionalTypePayment) Write(writer io.Writer, value *Payment) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypePaymentINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypePayment struct{}

func (_ FfiDestroyerOptionalTypePayment) Destroy(value *Payment) {
	if value != nil {
		FfiDestroyerTypePayment{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSymbol struct{}

var FfiConverterOptionalTypeSymbolINSTANCE = FfiConverterOptionalTypeSymbol{}

func (c FfiConverterOptionalTypeSymbol) Lift(rb RustBufferI) *Symbol {
	return LiftFromRustBuffer[*Symbol](c, rb)
}

func (_ FfiConverterOptionalTypeSymbol) Read(reader io.Reader) *Symbol {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSymbolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSymbol) Lower(value *Symbol) RustBuffer {
	return LowerIntoRustBuffer[*Symbol](c, value)
}

func (_ FfiConverterOptionalTypeSymbol) Write(writer io.Writer, value *Symbol) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSymbolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSymbol struct{}

func (_ FfiDestroyerOptionalTypeSymbol) Destroy(value *Symbol) {
	if value != nil {
		FfiDestroyerTypeSymbol{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeAmount struct{}

var FfiConverterOptionalTypeAmountINSTANCE = FfiConverterOptionalTypeAmount{}

func (c FfiConverterOptionalTypeAmount) Lift(rb RustBufferI) *Amount {
	return LiftFromRustBuffer[*Amount](c, rb)
}

func (_ FfiConverterOptionalTypeAmount) Read(reader io.Reader) *Amount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeAmount) Lower(value *Amount) RustBuffer {
	return LowerIntoRustBuffer[*Amount](c, value)
}

func (_ FfiConverterOptionalTypeAmount) Write(writer io.Writer, value *Amount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeAmount struct{}

func (_ FfiDestroyerOptionalTypeAmount) Destroy(value *Amount) {
	if value != nil {
		FfiDestroyerTypeAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeListPaymentDetails struct{}

var FfiConverterOptionalTypeListPaymentDetailsINSTANCE = FfiConverterOptionalTypeListPaymentDetails{}

func (c FfiConverterOptionalTypeListPaymentDetails) Lift(rb RustBufferI) *ListPaymentDetails {
	return LiftFromRustBuffer[*ListPaymentDetails](c, rb)
}

func (_ FfiConverterOptionalTypeListPaymentDetails) Read(reader io.Reader) *ListPaymentDetails {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeListPaymentDetailsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeListPaymentDetails) Lower(value *ListPaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[*ListPaymentDetails](c, value)
}

func (_ FfiConverterOptionalTypeListPaymentDetails) Write(writer io.Writer, value *ListPaymentDetails) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeListPaymentDetailsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeListPaymentDetails struct{}

func (_ FfiDestroyerOptionalTypeListPaymentDetails) Destroy(value *ListPaymentDetails) {
	if value != nil {
		FfiDestroyerTypeListPaymentDetails{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypePayAmount struct{}

var FfiConverterOptionalTypePayAmountINSTANCE = FfiConverterOptionalTypePayAmount{}

func (c FfiConverterOptionalTypePayAmount) Lift(rb RustBufferI) *PayAmount {
	return LiftFromRustBuffer[*PayAmount](c, rb)
}

func (_ FfiConverterOptionalTypePayAmount) Read(reader io.Reader) *PayAmount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypePayAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypePayAmount) Lower(value *PayAmount) RustBuffer {
	return LowerIntoRustBuffer[*PayAmount](c, value)
}

func (_ FfiConverterOptionalTypePayAmount) Write(writer io.Writer, value *PayAmount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypePayAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypePayAmount struct{}

func (_ FfiDestroyerOptionalTypePayAmount) Destroy(value *PayAmount) {
	if value != nil {
		FfiDestroyerTypePayAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeReceiveAmount struct{}

var FfiConverterOptionalTypeReceiveAmountINSTANCE = FfiConverterOptionalTypeReceiveAmount{}

func (c FfiConverterOptionalTypeReceiveAmount) Lift(rb RustBufferI) *ReceiveAmount {
	return LiftFromRustBuffer[*ReceiveAmount](c, rb)
}

func (_ FfiConverterOptionalTypeReceiveAmount) Read(reader io.Reader) *ReceiveAmount {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeReceiveAmountINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeReceiveAmount) Lower(value *ReceiveAmount) RustBuffer {
	return LowerIntoRustBuffer[*ReceiveAmount](c, value)
}

func (_ FfiConverterOptionalTypeReceiveAmount) Write(writer io.Writer, value *ReceiveAmount) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeReceiveAmountINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeReceiveAmount struct{}

func (_ FfiDestroyerOptionalTypeReceiveAmount) Destroy(value *ReceiveAmount) {
	if value != nil {
		FfiDestroyerTypeReceiveAmount{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSuccessAction struct{}

var FfiConverterOptionalTypeSuccessActionINSTANCE = FfiConverterOptionalTypeSuccessAction{}

func (c FfiConverterOptionalTypeSuccessAction) Lift(rb RustBufferI) *SuccessAction {
	return LiftFromRustBuffer[*SuccessAction](c, rb)
}

func (_ FfiConverterOptionalTypeSuccessAction) Read(reader io.Reader) *SuccessAction {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSuccessActionINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSuccessAction) Lower(value *SuccessAction) RustBuffer {
	return LowerIntoRustBuffer[*SuccessAction](c, value)
}

func (_ FfiConverterOptionalTypeSuccessAction) Write(writer io.Writer, value *SuccessAction) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSuccessActionINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSuccessAction struct{}

func (_ FfiDestroyerOptionalTypeSuccessAction) Destroy(value *SuccessAction) {
	if value != nil {
		FfiDestroyerTypeSuccessAction{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSuccessActionProcessed struct{}

var FfiConverterOptionalTypeSuccessActionProcessedINSTANCE = FfiConverterOptionalTypeSuccessActionProcessed{}

func (c FfiConverterOptionalTypeSuccessActionProcessed) Lift(rb RustBufferI) *SuccessActionProcessed {
	return LiftFromRustBuffer[*SuccessActionProcessed](c, rb)
}

func (_ FfiConverterOptionalTypeSuccessActionProcessed) Read(reader io.Reader) *SuccessActionProcessed {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSuccessActionProcessedINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSuccessActionProcessed) Lower(value *SuccessActionProcessed) RustBuffer {
	return LowerIntoRustBuffer[*SuccessActionProcessed](c, value)
}

func (_ FfiConverterOptionalTypeSuccessActionProcessed) Write(writer io.Writer, value *SuccessActionProcessed) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSuccessActionProcessedINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSuccessActionProcessed struct{}

func (_ FfiDestroyerOptionalTypeSuccessActionProcessed) Destroy(value *SuccessActionProcessed) {
	if value != nil {
		FfiDestroyerTypeSuccessActionProcessed{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypeAssetMetadata struct{}

var FfiConverterOptionalSequenceTypeAssetMetadataINSTANCE = FfiConverterOptionalSequenceTypeAssetMetadata{}

func (c FfiConverterOptionalSequenceTypeAssetMetadata) Lift(rb RustBufferI) *[]AssetMetadata {
	return LiftFromRustBuffer[*[]AssetMetadata](c, rb)
}

func (_ FfiConverterOptionalSequenceTypeAssetMetadata) Read(reader io.Reader) *[]AssetMetadata {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypeAssetMetadataINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypeAssetMetadata) Lower(value *[]AssetMetadata) RustBuffer {
	return LowerIntoRustBuffer[*[]AssetMetadata](c, value)
}

func (_ FfiConverterOptionalSequenceTypeAssetMetadata) Write(writer io.Writer, value *[]AssetMetadata) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypeAssetMetadataINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypeAssetMetadata struct{}

func (_ FfiDestroyerOptionalSequenceTypeAssetMetadata) Destroy(value *[]AssetMetadata) {
	if value != nil {
		FfiDestroyerSequenceTypeAssetMetadata{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypeExternalInputParser struct{}

var FfiConverterOptionalSequenceTypeExternalInputParserINSTANCE = FfiConverterOptionalSequenceTypeExternalInputParser{}

func (c FfiConverterOptionalSequenceTypeExternalInputParser) Lift(rb RustBufferI) *[]ExternalInputParser {
	return LiftFromRustBuffer[*[]ExternalInputParser](c, rb)
}

func (_ FfiConverterOptionalSequenceTypeExternalInputParser) Read(reader io.Reader) *[]ExternalInputParser {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypeExternalInputParserINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypeExternalInputParser) Lower(value *[]ExternalInputParser) RustBuffer {
	return LowerIntoRustBuffer[*[]ExternalInputParser](c, value)
}

func (_ FfiConverterOptionalSequenceTypeExternalInputParser) Write(writer io.Writer, value *[]ExternalInputParser) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypeExternalInputParserINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypeExternalInputParser struct{}

func (_ FfiDestroyerOptionalSequenceTypeExternalInputParser) Destroy(value *[]ExternalInputParser) {
	if value != nil {
		FfiDestroyerSequenceTypeExternalInputParser{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypePaymentState struct{}

var FfiConverterOptionalSequenceTypePaymentStateINSTANCE = FfiConverterOptionalSequenceTypePaymentState{}

func (c FfiConverterOptionalSequenceTypePaymentState) Lift(rb RustBufferI) *[]PaymentState {
	return LiftFromRustBuffer[*[]PaymentState](c, rb)
}

func (_ FfiConverterOptionalSequenceTypePaymentState) Read(reader io.Reader) *[]PaymentState {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypePaymentStateINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypePaymentState) Lower(value *[]PaymentState) RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentState](c, value)
}

func (_ FfiConverterOptionalSequenceTypePaymentState) Write(writer io.Writer, value *[]PaymentState) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypePaymentStateINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypePaymentState struct{}

func (_ FfiDestroyerOptionalSequenceTypePaymentState) Destroy(value *[]PaymentState) {
	if value != nil {
		FfiDestroyerSequenceTypePaymentState{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypePaymentType struct{}

var FfiConverterOptionalSequenceTypePaymentTypeINSTANCE = FfiConverterOptionalSequenceTypePaymentType{}

func (c FfiConverterOptionalSequenceTypePaymentType) Lift(rb RustBufferI) *[]PaymentType {
	return LiftFromRustBuffer[*[]PaymentType](c, rb)
}

func (_ FfiConverterOptionalSequenceTypePaymentType) Read(reader io.Reader) *[]PaymentType {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypePaymentTypeINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypePaymentType) Lower(value *[]PaymentType) RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentType](c, value)
}

func (_ FfiConverterOptionalSequenceTypePaymentType) Write(writer io.Writer, value *[]PaymentType) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypePaymentTypeINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypePaymentType struct{}

func (_ FfiDestroyerOptionalSequenceTypePaymentType) Destroy(value *[]PaymentType) {
	if value != nil {
		FfiDestroyerSequenceTypePaymentType{}.Destroy(*value)
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

func (c FfiConverterSequenceUint8) Lower(value []uint8) RustBuffer {
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

func (c FfiConverterSequenceString) Lower(value []string) RustBuffer {
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

type FfiConverterSequenceTypeAssetBalance struct{}

var FfiConverterSequenceTypeAssetBalanceINSTANCE = FfiConverterSequenceTypeAssetBalance{}

func (c FfiConverterSequenceTypeAssetBalance) Lift(rb RustBufferI) []AssetBalance {
	return LiftFromRustBuffer[[]AssetBalance](c, rb)
}

func (c FfiConverterSequenceTypeAssetBalance) Read(reader io.Reader) []AssetBalance {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]AssetBalance, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeAssetBalanceINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeAssetBalance) Lower(value []AssetBalance) RustBuffer {
	return LowerIntoRustBuffer[[]AssetBalance](c, value)
}

func (c FfiConverterSequenceTypeAssetBalance) Write(writer io.Writer, value []AssetBalance) {
	if len(value) > math.MaxInt32 {
		panic("[]AssetBalance is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeAssetBalanceINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeAssetBalance struct{}

func (FfiDestroyerSequenceTypeAssetBalance) Destroy(sequence []AssetBalance) {
	for _, value := range sequence {
		FfiDestroyerTypeAssetBalance{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeAssetMetadata struct{}

var FfiConverterSequenceTypeAssetMetadataINSTANCE = FfiConverterSequenceTypeAssetMetadata{}

func (c FfiConverterSequenceTypeAssetMetadata) Lift(rb RustBufferI) []AssetMetadata {
	return LiftFromRustBuffer[[]AssetMetadata](c, rb)
}

func (c FfiConverterSequenceTypeAssetMetadata) Read(reader io.Reader) []AssetMetadata {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]AssetMetadata, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeAssetMetadataINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeAssetMetadata) Lower(value []AssetMetadata) RustBuffer {
	return LowerIntoRustBuffer[[]AssetMetadata](c, value)
}

func (c FfiConverterSequenceTypeAssetMetadata) Write(writer io.Writer, value []AssetMetadata) {
	if len(value) > math.MaxInt32 {
		panic("[]AssetMetadata is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeAssetMetadataINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeAssetMetadata struct{}

func (FfiDestroyerSequenceTypeAssetMetadata) Destroy(sequence []AssetMetadata) {
	for _, value := range sequence {
		FfiDestroyerTypeAssetMetadata{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeExternalInputParser struct{}

var FfiConverterSequenceTypeExternalInputParserINSTANCE = FfiConverterSequenceTypeExternalInputParser{}

func (c FfiConverterSequenceTypeExternalInputParser) Lift(rb RustBufferI) []ExternalInputParser {
	return LiftFromRustBuffer[[]ExternalInputParser](c, rb)
}

func (c FfiConverterSequenceTypeExternalInputParser) Read(reader io.Reader) []ExternalInputParser {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ExternalInputParser, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeExternalInputParserINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeExternalInputParser) Lower(value []ExternalInputParser) RustBuffer {
	return LowerIntoRustBuffer[[]ExternalInputParser](c, value)
}

func (c FfiConverterSequenceTypeExternalInputParser) Write(writer io.Writer, value []ExternalInputParser) {
	if len(value) > math.MaxInt32 {
		panic("[]ExternalInputParser is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeExternalInputParserINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeExternalInputParser struct{}

func (FfiDestroyerSequenceTypeExternalInputParser) Destroy(sequence []ExternalInputParser) {
	for _, value := range sequence {
		FfiDestroyerTypeExternalInputParser{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeFiatCurrency struct{}

var FfiConverterSequenceTypeFiatCurrencyINSTANCE = FfiConverterSequenceTypeFiatCurrency{}

func (c FfiConverterSequenceTypeFiatCurrency) Lift(rb RustBufferI) []FiatCurrency {
	return LiftFromRustBuffer[[]FiatCurrency](c, rb)
}

func (c FfiConverterSequenceTypeFiatCurrency) Read(reader io.Reader) []FiatCurrency {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]FiatCurrency, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeFiatCurrencyINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeFiatCurrency) Lower(value []FiatCurrency) RustBuffer {
	return LowerIntoRustBuffer[[]FiatCurrency](c, value)
}

func (c FfiConverterSequenceTypeFiatCurrency) Write(writer io.Writer, value []FiatCurrency) {
	if len(value) > math.MaxInt32 {
		panic("[]FiatCurrency is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeFiatCurrencyINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeFiatCurrency struct{}

func (FfiDestroyerSequenceTypeFiatCurrency) Destroy(sequence []FiatCurrency) {
	for _, value := range sequence {
		FfiDestroyerTypeFiatCurrency{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLnOfferBlindedPath struct{}

var FfiConverterSequenceTypeLnOfferBlindedPathINSTANCE = FfiConverterSequenceTypeLnOfferBlindedPath{}

func (c FfiConverterSequenceTypeLnOfferBlindedPath) Lift(rb RustBufferI) []LnOfferBlindedPath {
	return LiftFromRustBuffer[[]LnOfferBlindedPath](c, rb)
}

func (c FfiConverterSequenceTypeLnOfferBlindedPath) Read(reader io.Reader) []LnOfferBlindedPath {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LnOfferBlindedPath, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLnOfferBlindedPathINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLnOfferBlindedPath) Lower(value []LnOfferBlindedPath) RustBuffer {
	return LowerIntoRustBuffer[[]LnOfferBlindedPath](c, value)
}

func (c FfiConverterSequenceTypeLnOfferBlindedPath) Write(writer io.Writer, value []LnOfferBlindedPath) {
	if len(value) > math.MaxInt32 {
		panic("[]LnOfferBlindedPath is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLnOfferBlindedPathINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLnOfferBlindedPath struct{}

func (FfiDestroyerSequenceTypeLnOfferBlindedPath) Destroy(sequence []LnOfferBlindedPath) {
	for _, value := range sequence {
		FfiDestroyerTypeLnOfferBlindedPath{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLocaleOverrides struct{}

var FfiConverterSequenceTypeLocaleOverridesINSTANCE = FfiConverterSequenceTypeLocaleOverrides{}

func (c FfiConverterSequenceTypeLocaleOverrides) Lift(rb RustBufferI) []LocaleOverrides {
	return LiftFromRustBuffer[[]LocaleOverrides](c, rb)
}

func (c FfiConverterSequenceTypeLocaleOverrides) Read(reader io.Reader) []LocaleOverrides {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocaleOverrides, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLocaleOverridesINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLocaleOverrides) Lower(value []LocaleOverrides) RustBuffer {
	return LowerIntoRustBuffer[[]LocaleOverrides](c, value)
}

func (c FfiConverterSequenceTypeLocaleOverrides) Write(writer io.Writer, value []LocaleOverrides) {
	if len(value) > math.MaxInt32 {
		panic("[]LocaleOverrides is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLocaleOverridesINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLocaleOverrides struct{}

func (FfiDestroyerSequenceTypeLocaleOverrides) Destroy(sequence []LocaleOverrides) {
	for _, value := range sequence {
		FfiDestroyerTypeLocaleOverrides{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLocalizedName struct{}

var FfiConverterSequenceTypeLocalizedNameINSTANCE = FfiConverterSequenceTypeLocalizedName{}

func (c FfiConverterSequenceTypeLocalizedName) Lift(rb RustBufferI) []LocalizedName {
	return LiftFromRustBuffer[[]LocalizedName](c, rb)
}

func (c FfiConverterSequenceTypeLocalizedName) Read(reader io.Reader) []LocalizedName {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocalizedName, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLocalizedNameINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLocalizedName) Lower(value []LocalizedName) RustBuffer {
	return LowerIntoRustBuffer[[]LocalizedName](c, value)
}

func (c FfiConverterSequenceTypeLocalizedName) Write(writer io.Writer, value []LocalizedName) {
	if len(value) > math.MaxInt32 {
		panic("[]LocalizedName is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLocalizedNameINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLocalizedName struct{}

func (FfiDestroyerSequenceTypeLocalizedName) Destroy(sequence []LocalizedName) {
	for _, value := range sequence {
		FfiDestroyerTypeLocalizedName{}.Destroy(value)
	}
}

type FfiConverterSequenceTypePayment struct{}

var FfiConverterSequenceTypePaymentINSTANCE = FfiConverterSequenceTypePayment{}

func (c FfiConverterSequenceTypePayment) Lift(rb RustBufferI) []Payment {
	return LiftFromRustBuffer[[]Payment](c, rb)
}

func (c FfiConverterSequenceTypePayment) Read(reader io.Reader) []Payment {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Payment, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypePaymentINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypePayment) Lower(value []Payment) RustBuffer {
	return LowerIntoRustBuffer[[]Payment](c, value)
}

func (c FfiConverterSequenceTypePayment) Write(writer io.Writer, value []Payment) {
	if len(value) > math.MaxInt32 {
		panic("[]Payment is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypePaymentINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypePayment struct{}

func (FfiDestroyerSequenceTypePayment) Destroy(sequence []Payment) {
	for _, value := range sequence {
		FfiDestroyerTypePayment{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRate struct{}

var FfiConverterSequenceTypeRateINSTANCE = FfiConverterSequenceTypeRate{}

func (c FfiConverterSequenceTypeRate) Lift(rb RustBufferI) []Rate {
	return LiftFromRustBuffer[[]Rate](c, rb)
}

func (c FfiConverterSequenceTypeRate) Read(reader io.Reader) []Rate {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Rate, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRate) Lower(value []Rate) RustBuffer {
	return LowerIntoRustBuffer[[]Rate](c, value)
}

func (c FfiConverterSequenceTypeRate) Write(writer io.Writer, value []Rate) {
	if len(value) > math.MaxInt32 {
		panic("[]Rate is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRate struct{}

func (FfiDestroyerSequenceTypeRate) Destroy(sequence []Rate) {
	for _, value := range sequence {
		FfiDestroyerTypeRate{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRefundableSwap struct{}

var FfiConverterSequenceTypeRefundableSwapINSTANCE = FfiConverterSequenceTypeRefundableSwap{}

func (c FfiConverterSequenceTypeRefundableSwap) Lift(rb RustBufferI) []RefundableSwap {
	return LiftFromRustBuffer[[]RefundableSwap](c, rb)
}

func (c FfiConverterSequenceTypeRefundableSwap) Read(reader io.Reader) []RefundableSwap {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RefundableSwap, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRefundableSwapINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRefundableSwap) Lower(value []RefundableSwap) RustBuffer {
	return LowerIntoRustBuffer[[]RefundableSwap](c, value)
}

func (c FfiConverterSequenceTypeRefundableSwap) Write(writer io.Writer, value []RefundableSwap) {
	if len(value) > math.MaxInt32 {
		panic("[]RefundableSwap is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRefundableSwapINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRefundableSwap struct{}

func (FfiDestroyerSequenceTypeRefundableSwap) Destroy(sequence []RefundableSwap) {
	for _, value := range sequence {
		FfiDestroyerTypeRefundableSwap{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRouteHint struct{}

var FfiConverterSequenceTypeRouteHintINSTANCE = FfiConverterSequenceTypeRouteHint{}

func (c FfiConverterSequenceTypeRouteHint) Lift(rb RustBufferI) []RouteHint {
	return LiftFromRustBuffer[[]RouteHint](c, rb)
}

func (c FfiConverterSequenceTypeRouteHint) Read(reader io.Reader) []RouteHint {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHint, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRouteHintINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRouteHint) Lower(value []RouteHint) RustBuffer {
	return LowerIntoRustBuffer[[]RouteHint](c, value)
}

func (c FfiConverterSequenceTypeRouteHint) Write(writer io.Writer, value []RouteHint) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHint is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRouteHintINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRouteHint struct{}

func (FfiDestroyerSequenceTypeRouteHint) Destroy(sequence []RouteHint) {
	for _, value := range sequence {
		FfiDestroyerTypeRouteHint{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRouteHintHop struct{}

var FfiConverterSequenceTypeRouteHintHopINSTANCE = FfiConverterSequenceTypeRouteHintHop{}

func (c FfiConverterSequenceTypeRouteHintHop) Lift(rb RustBufferI) []RouteHintHop {
	return LiftFromRustBuffer[[]RouteHintHop](c, rb)
}

func (c FfiConverterSequenceTypeRouteHintHop) Read(reader io.Reader) []RouteHintHop {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHintHop, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRouteHintHopINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRouteHintHop) Lower(value []RouteHintHop) RustBuffer {
	return LowerIntoRustBuffer[[]RouteHintHop](c, value)
}

func (c FfiConverterSequenceTypeRouteHintHop) Write(writer io.Writer, value []RouteHintHop) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHintHop is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRouteHintHopINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRouteHintHop struct{}

func (FfiDestroyerSequenceTypeRouteHintHop) Destroy(sequence []RouteHintHop) {
	for _, value := range sequence {
		FfiDestroyerTypeRouteHintHop{}.Destroy(value)
	}
}

type FfiConverterSequenceTypePaymentState struct{}

var FfiConverterSequenceTypePaymentStateINSTANCE = FfiConverterSequenceTypePaymentState{}

func (c FfiConverterSequenceTypePaymentState) Lift(rb RustBufferI) []PaymentState {
	return LiftFromRustBuffer[[]PaymentState](c, rb)
}

func (c FfiConverterSequenceTypePaymentState) Read(reader io.Reader) []PaymentState {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentState, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypePaymentStateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypePaymentState) Lower(value []PaymentState) RustBuffer {
	return LowerIntoRustBuffer[[]PaymentState](c, value)
}

func (c FfiConverterSequenceTypePaymentState) Write(writer io.Writer, value []PaymentState) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentState is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypePaymentStateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypePaymentState struct{}

func (FfiDestroyerSequenceTypePaymentState) Destroy(sequence []PaymentState) {
	for _, value := range sequence {
		FfiDestroyerTypePaymentState{}.Destroy(value)
	}
}

type FfiConverterSequenceTypePaymentType struct{}

var FfiConverterSequenceTypePaymentTypeINSTANCE = FfiConverterSequenceTypePaymentType{}

func (c FfiConverterSequenceTypePaymentType) Lift(rb RustBufferI) []PaymentType {
	return LiftFromRustBuffer[[]PaymentType](c, rb)
}

func (c FfiConverterSequenceTypePaymentType) Read(reader io.Reader) []PaymentType {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentType, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypePaymentTypeINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypePaymentType) Lower(value []PaymentType) RustBuffer {
	return LowerIntoRustBuffer[[]PaymentType](c, value)
}

func (c FfiConverterSequenceTypePaymentType) Write(writer io.Writer, value []PaymentType) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentType is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypePaymentTypeINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypePaymentType struct{}

func (FfiDestroyerSequenceTypePaymentType) Destroy(sequence []PaymentType) {
	for _, value := range sequence {
		FfiDestroyerTypePaymentType{}.Destroy(value)
	}
}

func Connect(req ConnectRequest) (*BindingLiquidSdk, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_connect(FfiConverterTypeConnectRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BindingLiquidSdk
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBindingLiquidSdkINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ConnectWithSigner(req ConnectWithSignerRequest, signer Signer) (*BindingLiquidSdk, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_connect_with_signer(FfiConverterTypeConnectWithSignerRequestINSTANCE.Lower(req), FfiConverterCallbackInterfaceSignerINSTANCE.Lower(signer), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BindingLiquidSdk
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBindingLiquidSdkINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func DefaultConfig(network LiquidNetwork, breezApiKey *string) (Config, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_default_config(FfiConverterTypeLiquidNetworkINSTANCE.Lower(network), FfiConverterOptionalStringINSTANCE.Lower(breezApiKey), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue Config
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeConfigINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInvoice(input string) (LnInvoice, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_liquid_bindings_fn_func_parse_invoice(FfiConverterStringINSTANCE.Lower(input), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnInvoice
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLNInvoiceINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func SetLogger(logger Logger) error {
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_liquid_bindings_fn_func_set_logger(FfiConverterCallbackInterfaceLoggerINSTANCE.Lower(logger), _uniffiStatus)
		return false
	})
	return _uniffiErr
}
