package logger

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var kdlPool = buffer.NewPool()

// KDLEncoder encodes log entries in KDL format
type KDLEncoder struct {
	// Map to store encoded fields
	fields map[string]string
	mu     sync.Mutex
}

// NewKDLEncoder creates a new KDL encoder
func NewKDLEncoder() zapcore.Encoder {
	return &KDLEncoder{
		fields: make(map[string]string),
	}
}

// Clone creates a copy of the encoder
func (e *KDLEncoder) Clone() zapcore.Encoder {
	e.mu.Lock()
	defer e.mu.Unlock()

	fieldsCopy := make(map[string]string, len(e.fields))
	for k, v := range e.fields {
		fieldsCopy[k] = v
	}
	return &KDLEncoder{fields: fieldsCopy}
}

// EncodeEntry encodes a log entry and its fields to KDL format
func (e *KDLEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := kdlPool.Get()

	// Start with log level and message
	buf.AppendString(entry.Level.String())
	buf.AppendByte(' ')
	buf.AppendByte('"')
	buf.AppendString(entry.Message)
	buf.AppendByte('"')

	// Check if we need to add properties
	hasProps := !entry.Time.IsZero() || entry.LoggerName != "" || entry.Caller.Defined || entry.Stack != "" || len(fields) > 0

	if hasProps {
		buf.AppendString(" {\n")

		// Add timestamp
		if !entry.Time.IsZero() {
			buf.AppendString(`    timestamp="`)
			buf.AppendString(entry.Time.Format(time.RFC3339))
			buf.AppendString("\"\n")
		}

		// Add logger name
		if entry.LoggerName != "" {
			buf.AppendString(`    logger="`)
			buf.AppendString(entry.LoggerName)
			buf.AppendString("\"\n")
		}

		// Add caller
		if entry.Caller.Defined {
			buf.AppendString(`    file="`)
			buf.AppendString(entry.Caller.File)
			buf.AppendByte(':')
			buf.AppendString(fmt.Sprint(entry.Caller.Line))
			buf.AppendString("\"\n")
		}

		// Add stack
		if entry.Stack != "" {
			buf.AppendString(`    stack="`)
			buf.AppendString(escapeKDLString(entry.Stack))
			buf.AppendString("\"\n")
		}

		// Add fields from the encoder's stored fields
		e.mu.Lock()
		for key, value := range e.fields {
			buf.AppendString("    ")
			buf.AppendString(key)
			buf.AppendString(`="`)
			buf.AppendString(value)
			buf.AppendString("\"\n")
		}
		e.mu.Unlock()

		// Add all new fields
		for _, field := range fields {
			encodeFieldToKDL(buf, field)
		}

		buf.AppendByte('}')
	}

	buf.AppendString("\n")

	// Clear stored fields after encoding
	e.mu.Lock()
	e.fields = make(map[string]string)
	e.mu.Unlock()

	return buf, nil
}

// ObjectEncoder interface implementation
func (e *KDLEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// For arrays, use JSON representation
	var builder strings.Builder
	builder.WriteByte('[')
	err := marshaler.MarshalLogArray(&kdlArrayEncoder{w: &builder})
	builder.WriteByte(']')
	e.fields[key] = builder.String()
	return err
}

func (e *KDLEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// For objects, use JSON representation
	var builder strings.Builder
	builder.WriteByte('{')
	err := marshaler.MarshalLogObject(&kdlObjectEncoder{w: &builder})
	builder.WriteByte('}')
	e.fields[key] = builder.String()
	return err
}

func (e *KDLEncoder) AddBinary(key string, value []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%x", value)
}

func (e *KDLEncoder) AddByteString(key string, value []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = string(value)
}

func (e *KDLEncoder) AddBool(key string, value bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if value {
		e.fields[key] = "true"
	} else {
		e.fields[key] = "false"
	}
}

func (e *KDLEncoder) AddComplex128(key string, value complex128) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%v", value)
}

func (e *KDLEncoder) AddComplex64(key string, value complex64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%v", value)
}

func (e *KDLEncoder) AddDuration(key string, value time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = value.String()
}

func (e *KDLEncoder) AddFloat64(key string, value float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%f", value)
}

func (e *KDLEncoder) AddFloat32(key string, value float32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%f", value)
}

func (e *KDLEncoder) AddInt(key string, value int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddInt64(key string, value int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddInt32(key string, value int32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddInt16(key string, value int16) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddInt8(key string, value int8) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddString(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = value
}

func (e *KDLEncoder) AddTime(key string, value time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = value.Format(time.RFC3339)
}

func (e *KDLEncoder) AddUint(key string, value uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddUint64(key string, value uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddUint32(key string, value uint32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddUint16(key string, value uint16) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddUint8(key string, value uint8) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%d", value)
}

func (e *KDLEncoder) AddUintptr(key string, value uintptr) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%x", value)
}

func (e *KDLEncoder) AddReflected(key string, value interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fields[key] = fmt.Sprintf("%v", value)
	return nil
}

func (e *KDLEncoder) OpenNamespace(key string) {
	// Namespaces not supported in KDL output
	// Could add as a prefix to subsequent keys
}

// Helper types for array and object encoding
type kdlArrayEncoder struct {
	w *strings.Builder
}

func (e *kdlArrayEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	e.w.WriteString("[")
	err := arr.MarshalLogArray(e)
	e.w.WriteString("]")
	return err
}

func (e *kdlArrayEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	e.w.WriteString("{")
	err := obj.MarshalLogObject(&kdlObjectEncoder{w: e.w})
	e.w.WriteString("}")
	return err
}

func (e *kdlArrayEncoder) AppendBool(v bool) {
	if v {
		e.w.WriteString("true")
	} else {
		e.w.WriteString("false")
	}
}

func (e *kdlArrayEncoder) AppendByteString(v []byte) {
	e.w.Write(v)
}

func (e *kdlArrayEncoder) AppendComplex128(v complex128) {
	fmt.Fprintf(e.w, "%v", v)
}

func (e *kdlArrayEncoder) AppendComplex64(v complex64) {
	fmt.Fprintf(e.w, "%v", v)
}

func (e *kdlArrayEncoder) AppendDuration(v time.Duration) {
	e.w.WriteString(v.String())
}

func (e *kdlArrayEncoder) AppendFloat64(v float64) {
	fmt.Fprintf(e.w, "%f", v)
}

func (e *kdlArrayEncoder) AppendFloat32(v float32) {
	fmt.Fprintf(e.w, "%f", v)
}

func (e *kdlArrayEncoder) AppendInt(v int) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendInt64(v int64) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendInt32(v int32) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendInt16(v int16) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendInt8(v int8) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendString(v string) {
	fmt.Fprintf(e.w, "%q", v)
}

func (e *kdlArrayEncoder) AppendTime(v time.Time) {
	fmt.Fprintf(e.w, "%q", v.Format(time.RFC3339))
}

func (e *kdlArrayEncoder) AppendUint(v uint) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendUint64(v uint64) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendUint32(v uint32) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendUint16(v uint16) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendUint8(v uint8) {
	fmt.Fprintf(e.w, "%d", v)
}

func (e *kdlArrayEncoder) AppendUintptr(v uintptr) {
	fmt.Fprintf(e.w, "%x", v)
}

func (e *kdlArrayEncoder) AppendReflected(v interface{}) error {
	fmt.Fprintf(e.w, "%v", v)
	return nil
}

type kdlObjectEncoder struct {
	w *strings.Builder
}

func (e *kdlObjectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	fmt.Fprintf(e.w, "%s=[", key)
	err := marshaler.MarshalLogArray(&kdlArrayEncoder{w: e.w})
	e.w.WriteString("]")
	return err
}

func (e *kdlObjectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	fmt.Fprintf(e.w, "%s={", key)
	err := marshaler.MarshalLogObject(&kdlObjectEncoder{w: e.w})
	e.w.WriteString("}")
	return err
}

func (e *kdlObjectEncoder) AddBool(key string, v bool) {
	fmt.Fprintf(e.w, "%s=%v", key, v)
}

func (e *kdlObjectEncoder) AddBinary(key string, v []byte) {
	fmt.Fprintf(e.w, "%s=%x", key, v)
}

func (e *kdlObjectEncoder) AddByteString(key string, v []byte) {
	fmt.Fprintf(e.w, "%s=%s", key, string(v))
}

func (e *kdlObjectEncoder) AddComplex128(key string, v complex128) {
	fmt.Fprintf(e.w, "%s=%v", key, v)
}

func (e *kdlObjectEncoder) AddComplex64(key string, v complex64) {
	fmt.Fprintf(e.w, "%s=%v", key, v)
}

func (e *kdlObjectEncoder) AddDuration(key string, v time.Duration) {
	fmt.Fprintf(e.w, "%s=%s", key, v.String())
}

func (e *kdlObjectEncoder) AddFloat64(key string, v float64) {
	fmt.Fprintf(e.w, "%s=%f", key, v)
}

func (e *kdlObjectEncoder) AddFloat32(key string, v float32) {
	fmt.Fprintf(e.w, "%s=%f", key, v)
}

func (e *kdlObjectEncoder) AddInt(key string, v int) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddInt64(key string, v int64) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddInt32(key string, v int32) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddInt16(key string, v int16) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddInt8(key string, v int8) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddString(key, v string) {
	fmt.Fprintf(e.w, "%s=%q", key, v)
}

func (e *kdlObjectEncoder) AddTime(key string, v time.Time) {
	fmt.Fprintf(e.w, "%s=%q", key, v.Format(time.RFC3339))
}

func (e *kdlObjectEncoder) AddUint(key string, v uint) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddUint64(key string, v uint64) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddUint32(key string, v uint32) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddUint16(key string, v uint16) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddUint8(key string, v uint8) {
	fmt.Fprintf(e.w, "%s=%d", key, v)
}

func (e *kdlObjectEncoder) AddUintptr(key string, v uintptr) {
	fmt.Fprintf(e.w, "%s=%x", key, v)
}

func (e *kdlObjectEncoder) AddReflected(key string, v interface{}) error {
	fmt.Fprintf(e.w, "%s=%v", key, v)
	return nil
}

func (e *kdlObjectEncoder) OpenNamespace(key string) {
	// Namespaces not supported
}

// Helper function to encode a field to KDL
func encodeFieldToKDL(buf *buffer.Buffer, field zapcore.Field) {
	buf.AppendString("    ")
	buf.AppendString(field.Key)
	buf.AppendByte('=')

	switch field.Type {
	case zapcore.StringType:
		buf.AppendByte('"')
		buf.AppendString(escapeKDLString(field.String))
		buf.AppendByte('"')

	case zapcore.BoolType:
		if field.Integer == 1 {
			buf.AppendString("true")
		} else {
			buf.AppendString("false")
		}

	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		buf.AppendString(fmt.Sprint(field.Integer))

	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		buf.AppendString(fmt.Sprint(uint64(field.Integer)))

	case zapcore.Float64Type:
		// Convert int64 bits to float64
		f := float64frombits(uint64(field.Integer))
		buf.AppendString(fmt.Sprintf("%f", f))

	case zapcore.Float32Type:
		// Convert int64 bits to float32
		f32 := float32frombits(uint32(field.Integer))
		buf.AppendString(fmt.Sprintf("%f", f32))

	case zapcore.StringerType:
		if field.Interface != nil {
			buf.AppendByte('"')
			buf.AppendString(escapeKDLString(fmt.Sprint(field.Interface)))
			buf.AppendByte('"')
		}

	case zapcore.ErrorType:
		if err, ok := field.Interface.(error); ok && err != nil {
			buf.AppendByte('"')
			buf.AppendString(escapeKDLString(err.Error()))
			buf.AppendByte('"')
		}

	default:
		// For other types, use string representation
		buf.AppendByte('"')
		buf.AppendString(escapeKDLString(field.String))
		buf.AppendByte('"')
	}

	buf.AppendByte('\n')
}

// escapeKDLString escapes special characters in KDL strings
func escapeKDLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// Helper functions for float conversion using math package
func float64frombits(b uint64) float64 {
	return math.Float64frombits(b)
}

func float32frombits(b uint32) float32 {
	return math.Float32frombits(b)
}
