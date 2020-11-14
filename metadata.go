package nats

import (
	"github.com/project-flogo/core/data/coerce"
)

// Settings struct
type Settings struct {
	NatsClusterUrls          string `md:"natsClusterUrls,required"`
	NatsConnName             string `md:"natsConnName"`
	PayloadFormat						 string `md:"payloadFormat"`
	NatsUserName             string `md:"natsUserName"`
	NatsUserPassword         string `md:"natsUserPassword"`
	NatsToken                string `md:"natsToken"`
	NatsNkeySeedfile         string `md:"natsNkeySeedfile"`
	NatsCredentialFile       string `md:"natsCredentialFile"`
	AutoReconnect            bool   `md:"autoReconnect"`
	MaxReconnects            int    `md:"maxReconnects"`
	EnableRandomReconnection bool   `md:"enableRandomReconnection"`
	ReconnectWait            int    `md:"reconnectWait"`
	ReconnectBufferSize      int  `md:"reconnectBufferSize"`
	SkipVerify               bool   `md:"skipVerify"`
	CaFile                   string `md:"caFile"`
	CertFile                 string `md:"certFile"`
	KeyFile                  string `md:"keyFile"`
	EnableStreaming          bool   `md:"enableStreaming"`
	StanClusterID            string `md:"stanClusterID"`
}

// FromMap method of Settings
func (s *Settings) FromMap(values map[string]interface{}) error {

	var (
		err error
	)
	s.NatsClusterUrls, err = coerce.ToString(values["natsClusterUrls"])
	if err != nil {
		return err
	}

	s.NatsConnName, err = coerce.ToString(values["natsConnName"])
	if err != nil {
		return err
	}

	s.PayloadFormat, err = coerce.ToString(values["payloadFormat"])
	if err != nil {
		return err
	}

	s.NatsUserName, err = coerce.ToString(values["natsUserName"])
	if err != nil {
		return err
	}

	s.NatsUserPassword, err = coerce.ToString(values["natsUserPassword"])
	if err != nil {
		return err
	}

	s.NatsToken, err = coerce.ToString(values["natsToken"])
	if err != nil {
		return err
	}

	s.NatsNkeySeedfile, err = coerce.ToString(values["natsNkeySeedfile"])
	if err != nil {
		return err
	}

	s.NatsCredentialFile, err = coerce.ToString(values["natsCredentialFile"])
	if err != nil {
		return err
	}

	s.AutoReconnect, err = coerce.ToBool(values["autoReconnect"])
	if err != nil {
		return err
	}

	s.MaxReconnects, err = coerce.ToInt(values["maxReconnects"])
	if err != nil {
		return err
	}

	s.EnableRandomReconnection, err = coerce.ToBool(values["enableRandomReconnection"])
	if err != nil {
		return err
	}

	s.ReconnectWait, err = coerce.ToInt(values["reconnectWait"])
	if err != nil {
		return err
	}

	s.ReconnectBufferSize, err = coerce.ToInt(values["reconnectBufferSize"])
	if err != nil {
		return err
	}

	s.SkipVerify, err = coerce.ToBool(values["skipVerify"])
	if err != nil {
		return err
	}

	s.CaFile, err = coerce.ToString(values["caFile"])
	if err != nil {
		return err
	}

	s.CertFile, err = coerce.ToString(values["certFile"])
	if err != nil {
		return err
	}

	s.KeyFile, err = coerce.ToString(values["keyFile"])
	if err != nil {
		return err
	}

	s.EnableStreaming, err = coerce.ToBool(values["enableStreaming"])
	if err != nil {
		return err
	}

	s.StanClusterID, err = coerce.ToString(values["stanClusterID"])
	if err != nil {
		return err
	}

	return nil

}

// ToMap method of Settings
func (s *Settings) ToMap() map[string]interface{} {

	return map[string]interface{}{
		"natsClusterUrls": s.NatsClusterUrls,
		"natsConnName":    s.NatsConnName,
		"payloadFormat":    s.PayloadFormat,
		"natsUserName": s.NatsUserName,
		"natsUserPassword": s.NatsUserPassword,
		"natsToken": s.NatsToken,
		"natsNkeySeedfile": s.NatsNkeySeedfile,
		"natsCredentialFile": s.NatsNkeySeedfile,
		"autoReconnect": s.AutoReconnect,
		"maxReconnects": s.MaxReconnects,
		"enableRandomReconnection": s.EnableRandomReconnection,
		"reconnectWait": s.ReconnectWait,
		"reconnectBufferSize": s.ReconnectBufferSize,
		"skipVerify": s.SkipVerify,
		"caFile": s.CaFile,
		"certFile": s.CertFile,
		"keyFile": s.KeyFile,
		"enableStreaming": s.EnableStreaming,
		"stanClusterID": s.StanClusterID,
	}

}

// Input struct
type Input struct {
	Subject           string  `md:"subject,required"`
	ChannelID         string  `md:"channelId"`
	Data              string  `md:"data,required"`
	ReceivedTimestamp float64 `md:"receivedTimestmap"`
}

// FromMap method of Input
func (i *Input) FromMap(values map[string]interface{}) error {
	var err error

	i.Subject, err = coerce.ToString(values["subject"])
	if err != nil {
		return err
	}

	i.ChannelID, err = coerce.ToString(values["channelId"])
	if err != nil {
		return err
	}

	i.Data, err = coerce.ToString(values["data"])
	if err != nil {
		return err
	}

	if values["receivedTimestamp"] != nil {
		i.ReceivedTimestamp, err = coerce.ToFloat64(values["receivedTimestamp"])
		if err != nil {
			return err
		}
	}

	return nil
}

// ToMap method of Input
func (i *Input) ToMap() map[string]interface{} {
	dataMap := map[string]interface{}{
		"subject":   i.Subject,
		"channelId": i.ChannelID,
		"data":      i.Data,
	}
	if i.ReceivedTimestamp > float64(0) {
		dataMap["receivedTimestamp"] = i.ReceivedTimestamp
	}
	return dataMap
}

// Output struct
type Output struct {
	Status string                 `md:"status,required"`
	Result map[string]interface{} `md:"result"`
}

// FromMap method of Output
func (o *Output) FromMap(values map[string]interface{}) error {
	o.Status = values["status"].(string)
	o.Result = values["result"].(map[string]interface{})
	return nil
}

// ToMap method of Output
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"status": o.Status,
		"result": o.Result,
	}
}
