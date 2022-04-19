package rabbitrpc

import (
	"encoding/json"
)

// req res classification

type MethodCode int
type StatusCode int

const (
	MethodCodeGET MethodCode = iota
	MethodCodePOST
	MethodCodePUT
	MethodCodeDelete
)

const (
	StatusOK StatusCode = iota
	StatusError
)

// general data wrapper

type Envelope struct {
	Method MethodCode `json:"method"`
	Status StatusCode `json:"status"`

	FunctionToCall string `json:"function_to_call"`
	DataTypeName   string `json:"type_name"`
	Body           []byte `json:"body"`
}

func MakeBin(
	method MethodCode,
	status StatusCode,
	functionToCall string,
	dataTypeName string,
	dataPtr interface{},
) (binEnvelope []byte, errorJSONMarshaling *RabbitRPCError) {
	binData, err := json.Marshal(dataPtr)
	if err != nil {
		errorJSONMarshaling = &RabbitRPCError{
			What: err.Error(),
		}
		return
	}
	envelop := Envelope{
		Method:         method,
		Status:         status,
		FunctionToCall: functionToCall,
		DataTypeName:   dataTypeName,
		Body:           binData,
	}
	binEnvelope, err = json.Marshal(envelop)
	if err != nil {
		errorJSONMarshaling = &RabbitRPCError{
			What: err.Error(),
		}
	}
	return
}

func FromBin(bin []byte,
) (envelop *Envelope, errorJSONUnmarshaling *RabbitRPCError) {
	envelop = &Envelope{}
	err := json.Unmarshal(bin, envelop)
	if err != nil {
		errorJSONUnmarshaling = &RabbitRPCError{
			What: err.Error(),
		}
	}
	return
}

func (envelop *Envelope) Extract(dataPtr interface{},
) (errorJSONUnmarshaling *RabbitRPCError) {
	err := json.Unmarshal(envelop.Body, dataPtr)
	if err != nil {
		errorJSONUnmarshaling = &RabbitRPCError{
			What: err.Error(),
		}
	}
	return
}

// callback map for convenience of correlation id check

type CallbackPool map[string]func(raws Raws)

// error definitions

type RabbitRPCError struct {
	What string `json:"what"`
}

const ErrorTypeName = "RabbitRPCError"

func (err *RabbitRPCError) Error() string {
	return err.What
}

var ErrorTypeNotFound *RabbitRPCError = &RabbitRPCError{
	What: "type name is unknown",
}

var ErrorMethodCodeInvalid *RabbitRPCError = &RabbitRPCError{
	What: "method code is invalid",
}

var ErrorFunctionNotFound *RabbitRPCError = &RabbitRPCError{
	What: "function name is invalid",
}
