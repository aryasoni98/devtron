/*
Devtron Labs

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.
// NOTE : notnull and validate added manually, as auto-generation does not add notnull and validate.

package openapi

import (
	"encoding/json"
)

// UpdateApiTokenRequest struct for UpdateApiTokenRequest
type UpdateApiTokenRequest struct {
	// Description of api-token
	Description *string `json:"description,omitempty,notnull" validate:"required"`
	// Expiration time of api-token in milliseconds
	ExpireAtInMs *int64 `json:"expireAtInMs,omitempty"`
}

// NewUpdateApiTokenRequest instantiates a new UpdateApiTokenRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewUpdateApiTokenRequest() *UpdateApiTokenRequest {
	this := UpdateApiTokenRequest{}
	return &this
}

// NewUpdateApiTokenRequestWithDefaults instantiates a new UpdateApiTokenRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewUpdateApiTokenRequestWithDefaults() *UpdateApiTokenRequest {
	this := UpdateApiTokenRequest{}
	return &this
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *UpdateApiTokenRequest) GetDescription() string {
	if o == nil || o.Description == nil {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UpdateApiTokenRequest) GetDescriptionOk() (*string, bool) {
	if o == nil || o.Description == nil {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *UpdateApiTokenRequest) HasDescription() bool {
	if o != nil && o.Description != nil {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *UpdateApiTokenRequest) SetDescription(v string) {
	o.Description = &v
}

// GetExpireAtInMs returns the ExpireAtInMs field value if set, zero value otherwise.
func (o *UpdateApiTokenRequest) GetExpireAtInMs() int64 {
	if o == nil || o.ExpireAtInMs == nil {
		var ret int64
		return ret
	}
	return *o.ExpireAtInMs
}

// GetExpireAtInMsOk returns a tuple with the ExpireAtInMs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UpdateApiTokenRequest) GetExpireAtInMsOk() (*int64, bool) {
	if o == nil || o.ExpireAtInMs == nil {
		return nil, false
	}
	return o.ExpireAtInMs, true
}

// HasExpireAtInMs returns a boolean if a field has been set.
func (o *UpdateApiTokenRequest) HasExpireAtInMs() bool {
	if o != nil && o.ExpireAtInMs != nil {
		return true
	}

	return false
}

// SetExpireAtInMs gets a reference to the given int64 and assigns it to the ExpireAtInMs field.
func (o *UpdateApiTokenRequest) SetExpireAtInMs(v int64) {
	o.ExpireAtInMs = &v
}

func (o UpdateApiTokenRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Description != nil {
		toSerialize["description"] = o.Description
	}
	if o.ExpireAtInMs != nil {
		toSerialize["expireAtInMs"] = o.ExpireAtInMs
	}
	return json.Marshal(toSerialize)
}

type NullableUpdateApiTokenRequest struct {
	value *UpdateApiTokenRequest
	isSet bool
}

func (v NullableUpdateApiTokenRequest) Get() *UpdateApiTokenRequest {
	return v.value
}

func (v *NullableUpdateApiTokenRequest) Set(val *UpdateApiTokenRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableUpdateApiTokenRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableUpdateApiTokenRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableUpdateApiTokenRequest(val *UpdateApiTokenRequest) *NullableUpdateApiTokenRequest {
	return &NullableUpdateApiTokenRequest{value: val, isSet: true}
}

func (v NullableUpdateApiTokenRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableUpdateApiTokenRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

