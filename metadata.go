package nats

import (
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/property"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/data/coerce"
)

// Settings struct
type Settings struct {
	ClusterUrls string                 `md:"clusterUrls,required"` // The NATS cluster to connect to
	ConnName    string                 `md:"connName"`
	Auth        map[string]interface{} `md:"auth"`      // Auth setting
	Reconnect   map[string]interface{} `md:"reconnect"` // Reconnect setting
	SslConfig   map[string]interface{} `md:"sslConfig"` // SSL config setting
	Streaming   map[string]interface{} `md:"streaming"` // NATS streaming config
	DataType    string                 `md:"dataType"`  // Data type
}

var resolver = resolve.NewCompositeResolver(map[string]resolve.Resolver{
	".":        &resolve.ScopeResolver{},
	"env":      &resolve.EnvResolver{},
	"property": &property.Resolver{},
	"loop":     &resolve.LoopResolver{},
})

// FromMap method of Settings
func (s *Settings) FromMap(values map[string]interface{}) error {

	var (
		err error
	)
	s.ClusterUrls, err = coerce.ToString(values["clusterUrls"])
	if err != nil {
		return err
	}

	s.ConnName, err = coerce.ToString(values["connName"])
	if err != nil {
		return err
	}

	s.DataType, err = coerce.ToString(values["dataType"])
	if err != nil {
		return err
	}

	mapperFactory := mapper.NewFactory(resolver)

	if values["auth"] != nil {
		var authOptions map[string]interface{}
		authOptions, err = coerce.ToObject(values["auth"])
		if err != nil {
			return err
		}

		var authMapper mapper.Mapper
		authMapper, err = mapperFactory.NewMapper(authOptions)
		if err != nil {
			return err
		}
	
		var authValue map[string]interface{}
		authValue, err = authMapper.Apply(nil)
		if err != nil {
			return err
		}
		s.Auth = authValue
	}

	if values["reconnect"] != nil {
		var reconnectOptions map[string]interface{}
		reconnectOptions, err = coerce.ToObject(values["reconnect"])
		if err != nil {
			return err
		}

		var reconnectMapper mapper.Mapper
		reconnectMapper, err = mapperFactory.NewMapper(reconnectOptions)
		if err != nil {
			return err
		}
	
		var reconnectValue map[string]interface{}
		reconnectValue, err = reconnectMapper.Apply(nil)
		if err != nil {
			return err
		}
		s.Reconnect = reconnectValue
	}

	if values["sslConfig"] != nil {
		var sslConfigOptions map[string]interface{}
		sslConfigOptions, err = coerce.ToObject(values["sslConfig"])
		if err != nil {
			return err
		}

		var sslConfigMapper mapper.Mapper
		sslConfigMapper, err = mapperFactory.NewMapper(sslConfigOptions)
		if err != nil {
			return err
		}
	
		var sslConfigValue map[string]interface{}
		sslConfigValue, err = sslConfigMapper.Apply(nil)
		if err != nil {
			return err
		}
		s.SslConfig = sslConfigValue
	}

	if values["streaming"] != nil {
		var streamingOptions map[string]interface{}
		streamingOptions, err = coerce.ToObject(values["streaming"])
		if err != nil {
			return err
		}
		
		var streamingMapper mapper.Mapper
		streamingMapper, err = mapperFactory.NewMapper(streamingOptions)
		if err != nil {
			return err
		}
	
		var streamingValue map[string]interface{}
		streamingValue, err = streamingMapper.Apply(nil)
		if err != nil {
			return err
		}
		s.Streaming = streamingValue
	}

	return nil

}

// ToMap method of Settings
func (s *Settings) ToMap() map[string]interface{} {

	return map[string]interface{}{
		"clusterUrls": s.ClusterUrls,
		"connName":    s.ConnName,
		"dataType":    s.DataType,
		"auth":        s.Auth,
		"reconnect":   s.Reconnect,
		"sslConfig":   s.SslConfig,
		"streaming":   s.Streaming,
	}

}

// Input struct
type Input struct {
	Subject   string `md:"subject,required"`
	ChannelID string `md:"channelId"`
	Data      string `md:"data,required"`
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
