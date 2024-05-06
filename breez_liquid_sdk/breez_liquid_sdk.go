package breez_liquid_sdk

// #include <breez_liquid_sdk.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
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
		C.ffi_breez_liquid_sdk_bindings_rustbuffer_free(cb, status)
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
		return C.ffi_breez_liquid_sdk_bindings_rustbuffer_from_bytes(foreign, status)
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

	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 24
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_breez_liquid_sdk_bindings_uniffi_contract_version(uniffiStatus)
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("breez_liquid_sdk: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_func_connect(uniffiStatus)
		})
		if checksum != 5222 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_func_connect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_backup(uniffiStatus)
		})
		if checksum != 50666 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_get_info(uniffiStatus)
		})
		if checksum != 3659 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_get_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment(uniffiStatus)
		})
		if checksum != 45631 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_prepare_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_prepare_send_payment(uniffiStatus)
		})
		if checksum != 23943 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_prepare_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_receive_payment(uniffiStatus)
		})
		if checksum != 59000 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_restore(uniffiStatus)
		})
		if checksum != 37933 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_restore: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_send_payment(uniffiStatus)
		})
		if checksum != 61690 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_liquid_sdk: uniffi_breez_liquid_sdk_bindings_checksum_method_bindingliquidsdk_send_payment: UniFFI API checksum mismatch")
		}
	}
}

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

func (_self *BindingLiquidSdk) Backup() error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeLiquidSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_backup(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) GetInfo(req GetInfoRequest) (GetInfoResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLiquidSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_get_info(
			_pointer, FfiConverterTypeGetInfoRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue GetInfoResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeGetInfoResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareReceivePayment(req PrepareReceiveRequest) (PrepareReceiveResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_prepare_receive_payment(
			_pointer, FfiConverterTypePrepareReceiveRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareReceiveResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareReceiveResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) PrepareSendPayment(req PrepareSendRequest) (PrepareSendResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_prepare_send_payment(
			_pointer, FfiConverterTypePrepareSendRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareSendResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareSendResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) ReceivePayment(req PrepareReceiveResponse) (ReceivePaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_receive_payment(
			_pointer, FfiConverterTypePrepareReceiveResponseINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReceivePaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeReceivePaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BindingLiquidSdk) Restore(req RestoreRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeLiquidSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_restore(
			_pointer, FfiConverterTypeRestoreRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BindingLiquidSdk) SendPayment(req PrepareSendResponse) (SendPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BindingLiquidSdk")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_liquid_sdk_bindings_fn_method_bindingliquidsdk_send_payment(
			_pointer, FfiConverterTypePrepareSendResponseINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
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
				C.uniffi_breez_liquid_sdk_bindings_fn_free_bindingliquidsdk(pointer, status)
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

type ConnectRequest struct {
	Mnemonic string
	Network  Network
	DataDir  *string
}

func (r *ConnectRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Mnemonic)
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalString{}.Destroy(r.DataDir)
}

type FfiConverterTypeConnectRequest struct{}

var FfiConverterTypeConnectRequestINSTANCE = FfiConverterTypeConnectRequest{}

func (c FfiConverterTypeConnectRequest) Lift(rb RustBufferI) ConnectRequest {
	return LiftFromRustBuffer[ConnectRequest](c, rb)
}

func (c FfiConverterTypeConnectRequest) Read(reader io.Reader) ConnectRequest {
	return ConnectRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectRequest) Lower(value ConnectRequest) RustBuffer {
	return LowerIntoRustBuffer[ConnectRequest](c, value)
}

func (c FfiConverterTypeConnectRequest) Write(writer io.Writer, value ConnectRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Mnemonic)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DataDir)
}

type FfiDestroyerTypeConnectRequest struct{}

func (_ FfiDestroyerTypeConnectRequest) Destroy(value ConnectRequest) {
	value.Destroy()
}

type GetInfoRequest struct {
	WithScan bool
}

func (r *GetInfoRequest) Destroy() {
	FfiDestroyerBool{}.Destroy(r.WithScan)
}

type FfiConverterTypeGetInfoRequest struct{}

var FfiConverterTypeGetInfoRequestINSTANCE = FfiConverterTypeGetInfoRequest{}

func (c FfiConverterTypeGetInfoRequest) Lift(rb RustBufferI) GetInfoRequest {
	return LiftFromRustBuffer[GetInfoRequest](c, rb)
}

func (c FfiConverterTypeGetInfoRequest) Read(reader io.Reader) GetInfoRequest {
	return GetInfoRequest{
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGetInfoRequest) Lower(value GetInfoRequest) RustBuffer {
	return LowerIntoRustBuffer[GetInfoRequest](c, value)
}

func (c FfiConverterTypeGetInfoRequest) Write(writer io.Writer, value GetInfoRequest) {
	FfiConverterBoolINSTANCE.Write(writer, value.WithScan)
}

type FfiDestroyerTypeGetInfoRequest struct{}

func (_ FfiDestroyerTypeGetInfoRequest) Destroy(value GetInfoRequest) {
	value.Destroy()
}

type GetInfoResponse struct {
	BalanceSat uint64
	Pubkey     string
}

func (r *GetInfoResponse) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.BalanceSat)
	FfiDestroyerString{}.Destroy(r.Pubkey)
}

type FfiConverterTypeGetInfoResponse struct{}

var FfiConverterTypeGetInfoResponseINSTANCE = FfiConverterTypeGetInfoResponse{}

func (c FfiConverterTypeGetInfoResponse) Lift(rb RustBufferI) GetInfoResponse {
	return LiftFromRustBuffer[GetInfoResponse](c, rb)
}

func (c FfiConverterTypeGetInfoResponse) Read(reader io.Reader) GetInfoResponse {
	return GetInfoResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGetInfoResponse) Lower(value GetInfoResponse) RustBuffer {
	return LowerIntoRustBuffer[GetInfoResponse](c, value)
}

func (c FfiConverterTypeGetInfoResponse) Write(writer io.Writer, value GetInfoResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.BalanceSat)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
}

type FfiDestroyerTypeGetInfoResponse struct{}

func (_ FfiDestroyerTypeGetInfoResponse) Destroy(value GetInfoResponse) {
	value.Destroy()
}

type PrepareReceiveRequest struct {
	PayerAmountSat uint64
}

func (r *PrepareReceiveRequest) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.PayerAmountSat)
}

type FfiConverterTypePrepareReceiveRequest struct{}

var FfiConverterTypePrepareReceiveRequestINSTANCE = FfiConverterTypePrepareReceiveRequest{}

func (c FfiConverterTypePrepareReceiveRequest) Lift(rb RustBufferI) PrepareReceiveRequest {
	return LiftFromRustBuffer[PrepareReceiveRequest](c, rb)
}

func (c FfiConverterTypePrepareReceiveRequest) Read(reader io.Reader) PrepareReceiveRequest {
	return PrepareReceiveRequest{
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareReceiveRequest) Lower(value PrepareReceiveRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveRequest](c, value)
}

func (c FfiConverterTypePrepareReceiveRequest) Write(writer io.Writer, value PrepareReceiveRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.PayerAmountSat)
}

type FfiDestroyerTypePrepareReceiveRequest struct{}

func (_ FfiDestroyerTypePrepareReceiveRequest) Destroy(value PrepareReceiveRequest) {
	value.Destroy()
}

type PrepareReceiveResponse struct {
	PairHash       string
	PayerAmountSat uint64
	FeesSat        uint64
}

func (r *PrepareReceiveResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.PairHash)
	FfiDestroyerUint64{}.Destroy(r.PayerAmountSat)
	FfiDestroyerUint64{}.Destroy(r.FeesSat)
}

type FfiConverterTypePrepareReceiveResponse struct{}

var FfiConverterTypePrepareReceiveResponseINSTANCE = FfiConverterTypePrepareReceiveResponse{}

func (c FfiConverterTypePrepareReceiveResponse) Lift(rb RustBufferI) PrepareReceiveResponse {
	return LiftFromRustBuffer[PrepareReceiveResponse](c, rb)
}

func (c FfiConverterTypePrepareReceiveResponse) Read(reader io.Reader) PrepareReceiveResponse {
	return PrepareReceiveResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareReceiveResponse) Lower(value PrepareReceiveResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareReceiveResponse](c, value)
}

func (c FfiConverterTypePrepareReceiveResponse) Write(writer io.Writer, value PrepareReceiveResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.PairHash)
	FfiConverterUint64INSTANCE.Write(writer, value.PayerAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesSat)
}

type FfiDestroyerTypePrepareReceiveResponse struct{}

func (_ FfiDestroyerTypePrepareReceiveResponse) Destroy(value PrepareReceiveResponse) {
	value.Destroy()
}

type PrepareSendRequest struct {
	Invoice string
}

func (r *PrepareSendRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Invoice)
}

type FfiConverterTypePrepareSendRequest struct{}

var FfiConverterTypePrepareSendRequestINSTANCE = FfiConverterTypePrepareSendRequest{}

func (c FfiConverterTypePrepareSendRequest) Lift(rb RustBufferI) PrepareSendRequest {
	return LiftFromRustBuffer[PrepareSendRequest](c, rb)
}

func (c FfiConverterTypePrepareSendRequest) Read(reader io.Reader) PrepareSendRequest {
	return PrepareSendRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareSendRequest) Lower(value PrepareSendRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareSendRequest](c, value)
}

func (c FfiConverterTypePrepareSendRequest) Write(writer io.Writer, value PrepareSendRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerTypePrepareSendRequest struct{}

func (_ FfiDestroyerTypePrepareSendRequest) Destroy(value PrepareSendRequest) {
	value.Destroy()
}

type PrepareSendResponse struct {
	Id                string
	PayerAmountSat    uint64
	ReceiverAmountSat uint64
	TotalFees         uint64
	FundingAddress    string
	Invoice           string
}

func (r *PrepareSendResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerUint64{}.Destroy(r.PayerAmountSat)
	FfiDestroyerUint64{}.Destroy(r.ReceiverAmountSat)
	FfiDestroyerUint64{}.Destroy(r.TotalFees)
	FfiDestroyerString{}.Destroy(r.FundingAddress)
	FfiDestroyerString{}.Destroy(r.Invoice)
}

type FfiConverterTypePrepareSendResponse struct{}

var FfiConverterTypePrepareSendResponseINSTANCE = FfiConverterTypePrepareSendResponse{}

func (c FfiConverterTypePrepareSendResponse) Lift(rb RustBufferI) PrepareSendResponse {
	return LiftFromRustBuffer[PrepareSendResponse](c, rb)
}

func (c FfiConverterTypePrepareSendResponse) Read(reader io.Reader) PrepareSendResponse {
	return PrepareSendResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareSendResponse) Lower(value PrepareSendResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareSendResponse](c, value)
}

func (c FfiConverterTypePrepareSendResponse) Write(writer io.Writer, value PrepareSendResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterUint64INSTANCE.Write(writer, value.PayerAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.ReceiverAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalFees)
	FfiConverterStringINSTANCE.Write(writer, value.FundingAddress)
	FfiConverterStringINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerTypePrepareSendResponse struct{}

func (_ FfiDestroyerTypePrepareSendResponse) Destroy(value PrepareSendResponse) {
	value.Destroy()
}

type ReceivePaymentResponse struct {
	Id      string
	Invoice string
}

func (r *ReceivePaymentResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerString{}.Destroy(r.Invoice)
}

type FfiConverterTypeReceivePaymentResponse struct{}

var FfiConverterTypeReceivePaymentResponseINSTANCE = FfiConverterTypeReceivePaymentResponse{}

func (c FfiConverterTypeReceivePaymentResponse) Lift(rb RustBufferI) ReceivePaymentResponse {
	return LiftFromRustBuffer[ReceivePaymentResponse](c, rb)
}

func (c FfiConverterTypeReceivePaymentResponse) Read(reader io.Reader) ReceivePaymentResponse {
	return ReceivePaymentResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceivePaymentResponse) Lower(value ReceivePaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentResponse](c, value)
}

func (c FfiConverterTypeReceivePaymentResponse) Write(writer io.Writer, value ReceivePaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerTypeReceivePaymentResponse struct{}

func (_ FfiDestroyerTypeReceivePaymentResponse) Destroy(value ReceivePaymentResponse) {
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

type SendPaymentResponse struct {
	Txid string
}

func (r *SendPaymentResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Txid)
}

type FfiConverterTypeSendPaymentResponse struct{}

var FfiConverterTypeSendPaymentResponseINSTANCE = FfiConverterTypeSendPaymentResponse{}

func (c FfiConverterTypeSendPaymentResponse) Lift(rb RustBufferI) SendPaymentResponse {
	return LiftFromRustBuffer[SendPaymentResponse](c, rb)
}

func (c FfiConverterTypeSendPaymentResponse) Read(reader io.Reader) SendPaymentResponse {
	return SendPaymentResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendPaymentResponse) Lower(value SendPaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[SendPaymentResponse](c, value)
}

func (c FfiConverterTypeSendPaymentResponse) Write(writer io.Writer, value SendPaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Txid)
}

type FfiDestroyerTypeSendPaymentResponse struct{}

func (_ FfiDestroyerTypeSendPaymentResponse) Destroy(value SendPaymentResponse) {
	value.Destroy()
}

type LiquidSdkError struct {
	err error
}

func (err LiquidSdkError) Error() string {
	return fmt.Sprintf("LiquidSdkError: %s", err.err.Error())
}

func (err LiquidSdkError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLiquidSdkErrorGeneric = fmt.Errorf("LiquidSdkErrorGeneric")

// Variant structs
type LiquidSdkErrorGeneric struct {
	message string
}

func NewLiquidSdkErrorGeneric() *LiquidSdkError {
	return &LiquidSdkError{
		err: &LiquidSdkErrorGeneric{},
	}
}

func (err LiquidSdkErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self LiquidSdkErrorGeneric) Is(target error) bool {
	return target == ErrLiquidSdkErrorGeneric
}

type FfiConverterTypeLiquidSdkError struct{}

var FfiConverterTypeLiquidSdkErrorINSTANCE = FfiConverterTypeLiquidSdkError{}

func (c FfiConverterTypeLiquidSdkError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLiquidSdkError) Lower(value *LiquidSdkError) RustBuffer {
	return LowerIntoRustBuffer[*LiquidSdkError](c, value)
}

func (c FfiConverterTypeLiquidSdkError) Read(reader io.Reader) error {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &LiquidSdkError{&LiquidSdkErrorGeneric{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLiquidSdkError.Read()", errorID))
	}

}

func (c FfiConverterTypeLiquidSdkError) Write(writer io.Writer, value *LiquidSdkError) {
	switch variantValue := value.err.(type) {
	case *LiquidSdkErrorGeneric:
		writeInt32(writer, 1)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLiquidSdkError.Write", value))
	}
}

type Network uint

const (
	NetworkLiquid        Network = 1
	NetworkLiquidTestnet Network = 2
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
var ErrPaymentErrorAmountOutOfRange = fmt.Errorf("PaymentErrorAmountOutOfRange")
var ErrPaymentErrorAlreadyClaimed = fmt.Errorf("PaymentErrorAlreadyClaimed")
var ErrPaymentErrorGeneric = fmt.Errorf("PaymentErrorGeneric")
var ErrPaymentErrorInvalidInvoice = fmt.Errorf("PaymentErrorInvalidInvoice")
var ErrPaymentErrorInvalidPreimage = fmt.Errorf("PaymentErrorInvalidPreimage")
var ErrPaymentErrorLwkError = fmt.Errorf("PaymentErrorLwkError")
var ErrPaymentErrorPairsNotFound = fmt.Errorf("PaymentErrorPairsNotFound")
var ErrPaymentErrorPersistError = fmt.Errorf("PaymentErrorPersistError")
var ErrPaymentErrorSendError = fmt.Errorf("PaymentErrorSendError")
var ErrPaymentErrorSignerError = fmt.Errorf("PaymentErrorSignerError")

// Variant structs
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
		return &PaymentError{&PaymentErrorAmountOutOfRange{message}}
	case 2:
		return &PaymentError{&PaymentErrorAlreadyClaimed{message}}
	case 3:
		return &PaymentError{&PaymentErrorGeneric{message}}
	case 4:
		return &PaymentError{&PaymentErrorInvalidInvoice{message}}
	case 5:
		return &PaymentError{&PaymentErrorInvalidPreimage{message}}
	case 6:
		return &PaymentError{&PaymentErrorLwkError{message}}
	case 7:
		return &PaymentError{&PaymentErrorPairsNotFound{message}}
	case 8:
		return &PaymentError{&PaymentErrorPersistError{message}}
	case 9:
		return &PaymentError{&PaymentErrorSendError{message}}
	case 10:
		return &PaymentError{&PaymentErrorSignerError{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypePaymentError.Read()", errorID))
	}

}

func (c FfiConverterTypePaymentError) Write(writer io.Writer, value *PaymentError) {
	switch variantValue := value.err.(type) {
	case *PaymentErrorAmountOutOfRange:
		writeInt32(writer, 1)
	case *PaymentErrorAlreadyClaimed:
		writeInt32(writer, 2)
	case *PaymentErrorGeneric:
		writeInt32(writer, 3)
	case *PaymentErrorInvalidInvoice:
		writeInt32(writer, 4)
	case *PaymentErrorInvalidPreimage:
		writeInt32(writer, 5)
	case *PaymentErrorLwkError:
		writeInt32(writer, 6)
	case *PaymentErrorPairsNotFound:
		writeInt32(writer, 7)
	case *PaymentErrorPersistError:
		writeInt32(writer, 8)
	case *PaymentErrorSendError:
		writeInt32(writer, 9)
	case *PaymentErrorSignerError:
		writeInt32(writer, 10)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypePaymentError.Write", value))
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

func Connect(req ConnectRequest) (*BindingLiquidSdk, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLiquidSdkError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_liquid_sdk_bindings_fn_func_connect(FfiConverterTypeConnectRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BindingLiquidSdk
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBindingLiquidSdkINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}
