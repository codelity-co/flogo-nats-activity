package nats

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/property"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/log"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
)

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})
var resolver = resolve.NewCompositeResolver(map[string]resolve.Resolver{
	".":        &resolve.ScopeResolver{},
	"env":      &resolve.EnvResolver{},
	"property": &property.Resolver{},
	"loop":     &resolve.LoopResolver{},
})

func init() {
	_ = activity.Register(&Activity{}, New)
}

//New optional factory method, should be used if one activity instance per configuration is desired
func New(ctx activity.InitContext) (activity.Activity, error) {

	var (
		err error
		nc  *nats.Conn
	)
	logger := ctx.Logger()

	logger.Debug("Running New method of activity...")

	s := &Settings{}

	logger.Debug("Mapping Settings struct...")
	err = s.FromMap(ctx.Settings())
	if err != nil {
		logger.Errorf("Map settings error: %v", err)
		return nil, err
	}
	logger.Debug("Mapped Settings struct successfully")

	logger.Debugf("From Map Setting: %v", s)

	// Resolving auth settings
	if s.Auth != nil {
		ctx.Logger().Debugf("methodOpitons settings being resolved: %v", s.Auth)
		auth, err := resolveObject(s.Auth)
		if err != nil {
			return nil, err
		}
		s.Auth = auth
		ctx.Logger().Debugf("auth settings resolved: %v", s.Auth)
	}

	// Resolving reconnect settings
	if s.Reconnect != nil {
		ctx.Logger().Debugf("reconnect settings being resolved: %v", s.Reconnect)
		reconnect, err := resolveObject(s.Reconnect)
		if err != nil {
			return nil, err
		}
		s.Reconnect = reconnect
		ctx.Logger().Debugf("reconnect settings resolved: %v", s.Reconnect)
	}

	// Resolving sslConfig settings
	if s.SslConfig != nil {
		ctx.Logger().Debugf("sslConfig settings being resolved: %v", s.SslConfig)
		sslConfig, err := resolveObject(s.SslConfig)
		if err != nil {
			return nil, err
		}
		s.SslConfig = sslConfig
		ctx.Logger().Debugf("sslConfig settings resolved: %v", s.SslConfig)
	}

	// Resolving sslConfig settings
	if s.Streaming != nil {
		ctx.Logger().Debugf("streaming settings being resolved: %v", s.Streaming)
		streaming, err := resolveObject(s.Streaming)
		if err != nil {
			return nil, err
		}
		s.Streaming = streaming
		ctx.Logger().Debugf("streaming settings resolved: %v", s.Streaming)
	}

	logger.Debug("Getting NATS connection...")
	nc, err = getNatsConnection(logger, s)
	if err != nil {
		logger.Errorf("NATS connection error: %v", err)
		return nil, err
	}
	logger.Debug("Got NATS connection")

	logger.Debug("Creating Activity struct...")
	act := &Activity{
		activitySettings: s,
		logger:           logger,
		natsConn:         nc,
		natsStreaming:    false,
	}
	logger.Debug("Created Activity struct successfully")

	logger.Debugf("Streaming: %v", s.Streaming)

	if enableStreaming, ok := s.Streaming["enableStreaming"]; ok {
		logger.Debug("Enabling NATS streaming...")
		act.natsStreaming = enableStreaming.(bool)
		if act.natsStreaming {
			logger.Debug("Getting STAN connection...")
			act.stanConn, err = getStanConnection(logger, s.Streaming, nc)
			if err != nil {
				logger.Errorf("STAN connection error: %v", err)
				return nil, err
			}
			logger.Debug("Got STAN connection")
		}
		logger.Debug("Enabled NATS streaming successfully")
	}

	logger.Debug("Finished New method of activity")
	return act, nil
}

// Activity is an sample Activity that can be used as a base to create a custom activity
type Activity struct {
	activitySettings *Settings
	logger           log.Logger
	natsConn         *nats.Conn
	natsStreaming    bool
	stanConn         stan.Conn
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Logs the Message
func (a *Activity) Eval(ctx activity.Context) (bool, error) {

	var (
		err    error
		result map[string]interface{}
	)

	result = make(map[string]interface{})

	a.logger.Debug("Running Eval method of activity...")
	input := &Input{}

	a.logger.Debug("Getting Input object from context...")
	err = ctx.GetInputObject(input)
	if err != nil {
		a.logger.Errorf("Error getting Input object: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}
	a.logger.Debug("Got Input object successfully")
	a.logger.Debugf("Input: %v", input)

	payload := map[string]interface{}{
		"subject":            input.Subject,
		"message":            input.Data,
		"receivedTimestamp":  input.ReceivedTimestamp,
		"streamingTimestamp": float64(time.Now().UTC().UnixNano()) / float64(1000000),
	}
	var payloadBytes []byte
	payloadBytes, err = json.Marshal(payload)
	if err != nil {
		a.logger.Errorf("Marshal error: %v", err)
		return true, err
	}

	if !a.natsStreaming {

		a.logger.Debug("Publishing data to NATS subject...")
		if err = a.natsConn.Publish(input.Subject, payloadBytes); err != nil {
			a.logger.Errorf("Error publishing data to NATS subject: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}
		a.logger.Debug("Published data to NATS subject")

	} else {

		a.logger.Debug("Publishing data to STAN Channel...")
		result["ackedNuid"], err = a.stanConn.PublishAsync(input.ChannelID, payloadBytes, func(ackedNuid string, err error) {
			if err != nil {
				a.logger.Errorf("STAN acknowledgement error: %v", err)
			}
		})
		if err != nil {
			a.logger.Errorf("Error publishing data to STAN channel: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}
		a.logger.Debugf("Published data to STAN channel: %v", result)
	}

	err = a.OutputToContext(ctx, result, nil)
	if err != nil {
		a.logger.Errorf("Error setting output object in context: %v", err)
		return true, err
	}
	a.logger.Debug("Successfully set output object in context")

	return true, nil
}

func (a *Activity) OutputToContext(ctx activity.Context, result map[string]interface{}, err error) error {
	a.logger.Debug("Createing Ouptut struct...")
	var output *Output
	if err != nil {
		output = &Output{Status: "ERROR", Result: map[string]interface{}{"errorMessage": fmt.Sprintf("%v", err)}}
	} else {
		output = &Output{Status: "SUCCESS", Result: result}
	}
	a.logger.Debug("Setting output object in context...")
	return ctx.SetOutputObject(output)
}

func getNatsConnection(logger log.Logger, settings *Settings) (*nats.Conn, error) {
	var (
		err           error
		authOpts      []nats.Option
		reconnectOpts []nats.Option
		sslConfigOpts []nats.Option
		urlString     string
	)

	// Check ClusterUrls
	logger.Debug("Checking clusterUrls...")
	if err := checkClusterUrls(logger, settings); err != nil {
		logger.Errorf("Error checking clusterUrls: %v", err)
		return nil, err
	}
	logger.Debug("Checked")

	urlString = settings.ClusterUrls

	logger.Debug("Getting NATS connection auth settings...")
	authOpts, err = getNatsConnAuthOpts(logger, settings)
	if err != nil {
		logger.Errorf("Error getting NATS connection auth settings:: %v", err)
		return nil, err
	}
	logger.Debug("Got NATS connection auth settings")

	logger.Debug("Getting NATS connection reconnect settings...")
	reconnectOpts, err = getNatsConnReconnectOpts(logger, settings)
	if err != nil {
		logger.Errorf("Error getting NATS connection reconnect settings:: %v", err)
		return nil, err
	}
	logger.Debug("Got NATS connection reconnect settings")

	logger.Debug("Getting NATS connection sslConfig settings...")
	sslConfigOpts, err = getNatsConnSslConfigOpts(logger, settings)
	if err != nil {
		logger.Errorf("Error getting NATS connection sslConfig settings:: %v", err)
		return nil, err
	}
	logger.Debug("Got NATS connection sslConfig settings")

	natsOptions := append(authOpts, reconnectOpts...)
	natsOptions = append(natsOptions, sslConfigOpts...)

	// Check ConnName
	if len(settings.ConnName) > 0 {
		natsOptions = append(natsOptions, nats.Name(settings.ConnName))
	}

	return nats.Connect(urlString, natsOptions...)

}

// checkClusterUrls is function to all valid NATS cluster urls
func checkClusterUrls(logger log.Logger, settings *Settings) error {
	// Check ClusterUrls
	clusterUrls := strings.Split(settings.ClusterUrls, ",")
	logger.Debugf("clusterUrls: %v", clusterUrls)
	if len(clusterUrls) < 1 {
		return fmt.Errorf("ClusterUrl [%v] is invalid, require at least one url", settings.ClusterUrls)
	}
	for _, v := range clusterUrls {
		logger.Debugf("v: %v", v)
		if err := validateClusterURL(v); err != nil {
			return err
		}
	}
	return nil
}

// validateClusterUrl is function to check NATS cluster url specificaiton
func validateClusterURL(url string) error {
	hostPort := strings.Split(url, ":")
	if len(hostPort) < 2 || len(hostPort) > 3 {
		return fmt.Errorf("ClusterUrl must be composed of sections like \"{nats|tls}://host[:port]\"")
	}
	if len(hostPort) == 3 {
		i, err := strconv.Atoi(hostPort[2])
		if err != nil || i < 0 || i > 32767 {
			return fmt.Errorf("port specification [%v] is not numeric and between 0 and 32767", hostPort[2])
		}
	}
	if (hostPort[0] != "nats") && (hostPort[0] != "tls") {
		return fmt.Errorf("protocol schema [%v] is not nats or tls", hostPort[0])
	}

	return nil
}

// getNatsConnAuthOps return slice of nats.Option specific for NATS authentication
func getNatsConnAuthOpts(logger log.Logger, settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)
	// Check auth setting
	logger.Debugf("settings.Auth: %v", settings.Auth)
	if settings.Auth != nil && len(settings.Auth) > 0 {
		if username, ok := settings.Auth["username"]; ok { // Check if usename is defined
			password, ok := settings.Auth["password"] // check if password is defined
			if !ok {
				return nil, fmt.Errorf("Missing password")
			} else {
				// Create UserInfo NATS option
				opts = append(opts, nats.UserInfo(username.(string), password.(string)))
			}
		} else if token, ok := settings.Auth["token"]; ok { // Check if token is defined
			opts = append(opts, nats.Token(token.(string)))
		} else if nkeySeedfile, ok := settings.Auth["nkeySeedfile"]; ok { // Check if nkey seed file is defined
			nkey, err := nats.NkeyOptionFromSeed(nkeySeedfile.(string))
			if err != nil {
				return nil, err
			}
			opts = append(opts, nkey)
		} else if credfile, ok := settings.Auth["credfile"]; ok { // Check if credential file is defined
			opts = append(opts, nats.UserCredentials(credfile.(string)))
		}
	}
	return opts, nil
}

func getNatsConnReconnectOpts(logger log.Logger, settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)
	// Check reconnect setting
	logger.Debugf("settings.Reconnect: %v", settings.Reconnect)
	if settings.Reconnect != nil && len(settings.Reconnect) > 0 {

		// Enable autoReconnect
		if autoReconnect, ok := settings.Reconnect["autoReconnect"]; ok {
			if !autoReconnect.(bool) {
				opts = append(opts, nats.NoReconnect())
			}
		}

		// Max reconnect attempts
		if maxReconnects, ok := settings.Reconnect["maxReconnects"]; ok {
			opts = append(opts, nats.MaxReconnects(maxReconnects.(int64)))
		}

		// Don't randomize
		if dontRandomize, ok := settings.Reconnect["dontRandomize"]; ok {
			if dontRandomize.(bool) {
				opts = append(opts, nats.DontRandomize())
			}
		}

		// Reconnect wait in seconds
		if reconnectWait, ok := settings.Reconnect["reconnectWait"]; ok {
			duration, err := time.ParseDuration(fmt.Sprintf("%vs", reconnectWait))
			if err != nil {
				return nil, err
			}
			opts = append(opts, nats.ReconnectWait(duration))
		}

		// Reconnect buffer size in bytes
		if reconnectBufSize, ok := settings.Reconnect["reconnectBufSize"]; ok {
			opts = append(opts, nats.ReconnectBufSize(reconnectBufSize.(int64)))
		}
	}
	return opts, nil
}

func getNatsConnSslConfigOpts(logger log.Logger, settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	// Check sslConfig setting
	logger.Debugf("settings.SslConfig: %v", settings.SslConfig)
	if settings.SslConfig != nil && len(settings.SslConfig) > 0 {
		// Skip verify
		if skipVerify, ok := settings.SslConfig["skipVerify"]; ok {
			opts = append(opts, nats.Secure(&tls.Config{
				InsecureSkipVerify: skipVerify.(bool),
			}))
		}

		// CA Root
		if caFile, ok := settings.SslConfig["caFile"]; ok {
			opts = append(opts, nats.RootCAs(caFile.(string)))
			// Cert file
			if certFile, ok := settings.SslConfig["certFile"]; ok {
				if keyFile, ok := settings.SslConfig["keyFile"]; ok {
					opts = append(opts, nats.ClientCert(certFile.(string), keyFile.(string)))
				} else {
					return nil, fmt.Errorf("Missing keyFile setting")
				}
			} else {
				return nil, fmt.Errorf("Missing certFile setting")
			}
		} else {
			return nil, fmt.Errorf("Missing caFile setting")
		}
		

	}
	return opts, nil
}

func getStanConnection(logger log.Logger, mapping map[string]interface{}, conn *nats.Conn) (stan.Conn, error) {

	var (
		err       error
		clusterID string
		ok        bool
		hostname  string
		sc        stan.Conn
	)

	if _, ok = mapping["clusterId"]; !ok {
		return nil, fmt.Errorf("clusterId not found")
	}

	clusterID = mapping["clusterId"].(string)
	logger.Debugf("clusterID: %v", clusterID)
	hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = strings.Split(hostname, ".")[0]
	hostname = strings.Split(hostname, ":")[0]
	logger.Debugf("hostname: %v", hostname)
	logger.Debugf("natsConn: %v", conn)

	sc, err = stan.Connect(clusterID, hostname, stan.NatsConn(conn))
	if err != nil {
		return nil, err
	}

	return sc, nil
}

func resolveObject(object map[string]interface{}) (map[string]interface{}, error) {
	var err error

	mapperFactory := mapper.NewFactory(resolver)
	valuesMapper, err := mapperFactory.NewMapper(object)
	if err != nil {
		return nil, err
	}

	objectValues, err := valuesMapper.Apply(data.NewSimpleScope(map[string]interface{}{}, nil))
	if err != nil {
		return nil, err
	}

	return objectValues, nil
}
