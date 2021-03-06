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
	// "github.com/project-flogo/core/data"
	// "github.com/project-flogo/core/data/mapper"
	// "github.com/project-flogo/core/data/property"
	// "github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/log"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
)

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})
// var resolver = resolve.NewCompositeResolver(map[string]resolve.Resolver{
// 	".":        &resolve.ScopeResolver{},
// 	"env":      &resolve.EnvResolver{},
// 	"property": &property.Resolver{},
// 	"loop":     &resolve.LoopResolver{},
// })

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
	// if s.Auth != nil {
	// 	ctx.Logger().Debugf("auth settings being resolved: %v", s.Auth)
	// 	auth, err := resolveObject(s.Auth)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	s.Auth = auth
	// 	ctx.Logger().Debugf("auth settings resolved: %v", s.Auth)
	// }

	// Resolving reconnect settings
	// if s.Reconnect != nil {
	// 	ctx.Logger().Debugf("reconnect settings being resolved: %v", s.Reconnect)
	// 	reconnect, err := resolveObject(s.Reconnect)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	s.Reconnect = reconnect
	// 	ctx.Logger().Debugf("reconnect settings resolved: %v", s.Reconnect)
	// }

	// Resolving sslConfig settings
	// if s.SslConfig != nil {
	// 	ctx.Logger().Debugf("sslConfig settings being resolved: %v", s.SslConfig)
	// 	sslConfig, err := resolveObject(s.SslConfig)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	s.SslConfig = sslConfig
	// 	ctx.Logger().Debugf("sslConfig settings resolved: %v", s.SslConfig)
	// }

	// Resolving streaming settings
	// if s.Streaming != nil {
	// 	ctx.Logger().Debugf("streaming settings being resolved: %v", s.Streaming)
	// 	streaming, err := resolveObject(s.Streaming)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	s.Streaming = streaming
	// 	ctx.Logger().Debugf("streaming settings resolved: %v", s.Streaming)
	// }

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
	}
	logger.Debug("Created Activity struct successfully")

	logger.Debugf("Enable Streaming: %v", s.EnableStreaming)
	logger.Debugf("Stan Cluster ID: %v", s.StanClusterID)

	if s.EnableStreaming {
		logger.Debug("Getting STAN connection...")
		act.stanConn, err = getStanConnection(logger, nc, s.StanClusterID)
		if err != nil {
			logger.Errorf("STAN connection error: %v", err)
			return nil, err
		}
		logger.Debug("Got STAN connection")

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
		"subject": input.Subject,
		"data": input.Data,
		"receivedTimestamp": input.ReceivedTimestamp,
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

	urlString = settings.NatsClusterUrls

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
	if len(settings.NatsConnName) > 0 {
		natsOptions = append(natsOptions, nats.Name(settings.NatsConnName))
	}

	return nats.Connect(urlString, natsOptions...)

}

// checkClusterUrls is function to all valid NATS cluster urls
func checkClusterUrls(logger log.Logger, settings *Settings) error {
	// Check ClusterUrls
	clusterUrls := strings.Split(settings.NatsClusterUrls, ",")
	logger.Debugf("clusterUrls: %v", clusterUrls)
	if len(clusterUrls) < 1 {
		return fmt.Errorf("ClusterUrl [%v] is invalid, require at least one url", settings.NatsClusterUrls)
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

	if settings.NatsUserName != "" { // Check if usename is defined
	  // check if password is defined
		if settings.NatsUserPassword == "" {
			return nil, fmt.Errorf("Missing password")
		} else {
			// Create UserInfo NATS option
			opts = append(opts, nats.UserInfo(settings.NatsUserName, settings.NatsUserPassword))
		}
	} else if settings.NatsToken != "" { // Check if token is defined
		opts = append(opts, nats.Token(settings.NatsToken))
	} else if settings.NatsNkeySeedfile != "" { // Check if nkey seed file is defined
		nkey, err := nats.NkeyOptionFromSeed(settings.NatsNkeySeedfile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, nkey)
	} else if settings.NatsCredentialFile != "" { // Check if credential file is defined
		opts = append(opts, nats.UserCredentials(settings.NatsCredentialFile))
	}
	return opts, nil
}

func getNatsConnReconnectOpts(logger log.Logger, settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	// Enable autoReconnect
	if !settings.AutoReconnect {
		opts = append(opts, nats.NoReconnect())
	}
	
	// Max reconnect attempts
	if settings.MaxReconnects > 0 {
		opts = append(opts, nats.MaxReconnects(settings.MaxReconnects))
	}

	// Don't randomize
	if settings.EnableRandomReconnection {
		opts = append(opts, nats.DontRandomize())
	}

	// Reconnect wait in seconds
	if settings.ReconnectWait > 0 {
		duration, err := time.ParseDuration(fmt.Sprintf("%vs", settings.ReconnectWait))
		if err != nil {
			return nil, err
		}
		opts = append(opts, nats.ReconnectWait(duration))
	}

	// Reconnect buffer size in bytes
	if settings.ReconnectBufferSize > 0 {
		opts = append(opts, nats.ReconnectBufSize(settings.ReconnectBufferSize))
	}
	return opts, nil
}

func getNatsConnSslConfigOpts(logger log.Logger, settings *Settings) ([]nats.Option, error) {
	opts := make([]nats.Option, 0)

	if settings.CertFile != "" && settings.KeyFile != "" {
		// Skip verify
		if settings.SkipVerify {
			opts = append(opts, nats.Secure(&tls.Config{
				InsecureSkipVerify: settings.SkipVerify,
			}))
		}
		// CA Root
		if settings.CaFile != "" {
			opts = append(opts, nats.RootCAs(settings.CaFile))
			// Cert file
			if settings.CertFile != "" {
				if settings.KeyFile != "" {
					opts = append(opts, nats.ClientCert(settings.CertFile, settings.KeyFile))
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

func getStanConnection(logger log.Logger, conn *nats.Conn, stanClusterID string) (stan.Conn, error) {

	if stanClusterID == "" {
		return nil, fmt.Errorf("missing stanClusterId")
	}

	logger.Debugf("clusterID: %v", stanClusterID)
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = strings.Split(hostname, ".")[0]
	hostname = strings.Split(hostname, ":")[0]
	logger.Debugf("hostname: %v", hostname)
	logger.Debugf("natsConn: %v", conn)

	sc, err := stan.Connect(stanClusterID, hostname, stan.NatsConn(conn))
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// func resolveObject(object map[string]interface{}) (map[string]interface{}, error) {
// 	var err error

// 	// mapperFactory := mapper.NewFactory(resolver)
// 	// valuesMapper, err := mapperFactory.NewMapper(object)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// objectValues, err := valuesMapper.Apply(data.NewSimpleScope(map[string]interface{}{}, nil))
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	return objectValues, nil
// }
