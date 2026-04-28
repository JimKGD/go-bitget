// Package ws provides WebSocket client functionality for real-time data streaming from Bitget.
// It supports both public market data and private authenticated channels with automatic
// reconnection, subscription management, and message handling.
//
// Example usage:
//
//	client := NewBitgetBaseWsClient(logger, "wss://ws.bitget.com/v2/ws/public", "")
//	client.SetListener(messageHandler, errorHandler)
//	client.Connect()
package ws

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/khanbekov/go-bitget/common"
	"github.com/khanbekov/go-bitget/common/types"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// OnReceive is a callback function type for handling incoming WebSocket messages.
// It receives the raw message string from the WebSocket connection.
type OnReceive func(message string)

// loginCredentials stores authentication details for re-authentication after reconnection
type loginCredentials struct {
	apiKey     string
	passphrase string
	signType   common.SignType
}

// rateLimiter implements rate limiting to prevent exceeding message send limits
type rateLimiter struct {
	lastSend    time.Time     // Timestamp of last message sent
	minInterval time.Duration // Minimum interval between messages (100ms for 10 messages/second)
	mutex       sync.Mutex    // Mutex for thread-safe access
}

// BaseWsClient provides a WebSocket client for Bitget's real-time API.
// It handles connection management, authentication, subscription tracking,
// and automatic reconnection with configurable timeouts.
//
// Concurrency: all exported methods are safe for concurrent use. The client
// uses two mutexes:
//   - subsMu guards the subscriptions map.
//   - stateMu guards connection state (connected, loginStatus, webSocketClient,
//     timestamps, reconnectAttempts, storedLoginCreds, needLogin, reconnecting).
//
// Never call a method that acquires stateMu while already holding stateMu.
type BaseWsClient struct {
	url           string         // WebSocket endpoint URL (immutable after construction)
	logger        zerolog.Logger // Logger for debugging and monitoring
	signer        *common.Signer // Signer for authentication
	listener      OnReceive      // Default message handler (set once before Connect)
	errorListener OnReceive      // Error message handler (set once before Connect)

	// Subscriptions state (guarded by subsMu)
	subsMu            sync.RWMutex
	subscriptions     map[SubscriptionArgs]OnReceive
	subscribeRequests *types.Set

	// Connection state (guarded by stateMu)
	stateMu               sync.Mutex
	needLogin             bool
	connected             bool
	loginStatus           bool
	webSocketClient       *websocket.Conn
	lastReceivedTime      time.Time
	connectionStartTime   time.Time
	reconnecting          bool
	reconnectAttempts     int
	storedLoginCreds      *loginCredentials
	checkConnectionTicker *time.Ticker
	reconnectionTimeout   time.Duration
	maxReconnectAttempts  int

	// Message send coordination
	sendMutex   *sync.Mutex
	rateLimiter *rateLimiter

	// Reconnection serialization (separate from stateMu to avoid holding stateMu across I/O)
	reconnectMutex sync.Mutex

	// Lifecycle management
	stopCh    chan struct{}
	closeOnce sync.Once
	pingCron  *cron.Cron
}

// NewBitgetBaseWsClient creates a new WebSocket client for Bitget's real-time API.
//
// Parameters:
//   - logger: Logger instance for debugging and monitoring
//   - url: WebSocket endpoint URL (public or private)
//   - secretKey: Secret key for authentication (empty string for public channels)
//
// Returns a configured BaseWsClient ready for connection.
func NewBitgetBaseWsClient(logger zerolog.Logger, url, secretKey string) *BaseWsClient {
	return &BaseWsClient{
		logger:                logger,
		url:                   url,
		subscribeRequests:     types.NewSet(),
		signer:                common.NewSigner(secretKey),
		subscriptions:         make(map[SubscriptionArgs]OnReceive),
		sendMutex:             &sync.Mutex{},
		checkConnectionTicker: time.NewTicker(5 * time.Second),
		reconnectionTimeout:   120 * time.Second,
		lastReceivedTime:      time.Now(),
		connectionStartTime:   time.Now(),
		maxReconnectAttempts:  5,
		rateLimiter: &rateLimiter{
			minInterval: 100 * time.Millisecond,
		},
		stopCh: make(chan struct{}),
	}
}

// SetCheckConnectionInterval configures how often the client checks connection health.
// Default is 5 seconds. Must be called before Connect.
func (c *BaseWsClient) SetCheckConnectionInterval(interval time.Duration) {
	c.stateMu.Lock()
	if c.checkConnectionTicker != nil {
		c.checkConnectionTicker.Stop()
	}
	c.checkConnectionTicker = time.NewTicker(interval)
	c.stateMu.Unlock()
}

// SetReconnectionTimeout sets how long to wait without receiving messages before reconnecting.
func (c *BaseWsClient) SetReconnectionTimeout(timeout time.Duration) {
	c.stateMu.Lock()
	c.reconnectionTimeout = timeout
	c.stateMu.Unlock()
}

// SetMaxReconnectAttempts sets the maximum number of reconnection attempts before giving up.
// Set to 0 for unlimited attempts.
func (c *BaseWsClient) SetMaxReconnectAttempts(maxAttempts int) {
	c.stateMu.Lock()
	c.maxReconnectAttempts = maxAttempts
	c.stateMu.Unlock()
}

// SetListener sets the default message and error handlers for the WebSocket client.
// Must be called before Connect (listeners are not protected by locks).
func (c *BaseWsClient) SetListener(msgListener OnReceive, errorListener OnReceive) {
	c.listener = msgListener
	c.errorListener = errorListener
}

// Connect initiates the WebSocket connection and starts the monitoring loop.
func (c *BaseWsClient) Connect() {
	go c.tickerLoop()
	if err := c.startPing(); err != nil {
		c.logger.Error().Err(err).Msg("fail to start ping")
	}
}

// ConnectWebSocket establishes the actual WebSocket connection to the Bitget server.
func (c *BaseWsClient) ConnectWebSocket() {
	c.logger.Info().Msg("WebSocket connecting...")
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		c.logger.Error().Err(err).Msg("WebSocket connection error")
		return
	}
	c.logger.Info().Msg("WebSocket connected")

	now := time.Now()
	c.stateMu.Lock()
	c.webSocketClient = conn
	c.connected = true
	c.connectionStartTime = now
	c.lastReceivedTime = now
	c.stateMu.Unlock()

	if c.subscriptionCount() > 0 {
		c.logger.Info().Int("subscription_count", c.subscriptionCount()).Msg("Restoring subscriptions after reconnection")
		c.restoreSubscriptions()
	}
}

// Login performs authentication for private WebSocket channels.
func (c *BaseWsClient) Login(apiKey, passphrase string, signType common.SignType) {
	c.stateMu.Lock()
	c.storedLoginCreds = &loginCredentials{
		apiKey:     apiKey,
		passphrase: passphrase,
		signType:   signType,
	}
	c.needLogin = true
	c.stateMu.Unlock()

	c.performLogin()
}

// performLogin executes the actual login process.
func (c *BaseWsClient) performLogin() {
	c.stateMu.Lock()
	creds := c.storedLoginCreds
	c.stateMu.Unlock()

	if creds == nil {
		c.logger.Error().Msg("No stored login credentials available")
		return
	}

	timestamp := common.TimestampSec()
	var sign string
	if creds.signType == common.SHA256 {
		sign = c.signer.Sign(common.WsAuthMethod, common.WsAuthPath, "", timestamp)
	} else {
		sign = c.signer.SignByRSA(common.WsAuthMethod, common.WsAuthPath, "", timestamp)
	}

	loginReq := WsLoginReq{
		ApiKey:     creds.apiKey,
		Passphrase: creds.passphrase,
		Timestamp:  timestamp,
		Sign:       sign,
	}
	baseReq := WsBaseReq{
		Op:   common.WsOpLogin,
		Args: []interface{}{loginReq},
	}
	c.SendByType(baseReq)
}

// StartReadLoop starts the background read loop.
func (c *BaseWsClient) StartReadLoop() {
	go c.ReadLoop()
}

func (c *BaseWsClient) startPing() error {
	cr := cron.New(cron.WithSeconds())
	if _, err := cr.AddFunc("*/30 * * * * *", c.ping); err != nil {
		return err
	}
	cr.Start()

	c.stateMu.Lock()
	c.pingCron = cr
	c.stateMu.Unlock()
	return nil
}

func (c *BaseWsClient) ping() {
	c.Send("ping")
}

func (c *BaseWsClient) SendByType(req WsBaseReq) {
	s, _ := jsoniter.MarshalToString(req)
	c.Send(s)
}

// Send sends a text message to the WebSocket server with rate limiting.
func (c *BaseWsClient) Send(data string) {
	c.stateMu.Lock()
	conn := c.webSocketClient
	c.stateMu.Unlock()

	if conn == nil {
		c.logger.Error().Msg("WebSocket sent error: no connection available")
		return
	}

	// Apply rate limiting (max 10 messages per second)
	c.rateLimiter.mutex.Lock()
	timeSinceLastSend := time.Since(c.rateLimiter.lastSend)
	if timeSinceLastSend < c.rateLimiter.minInterval {
		sleepDuration := c.rateLimiter.minInterval - timeSinceLastSend
		c.rateLimiter.mutex.Unlock()
		c.logger.Debug().Dur("sleep", sleepDuration).Msg("Rate limiting: sleeping before send")
		time.Sleep(sleepDuration)
		c.rateLimiter.mutex.Lock()
	}
	c.rateLimiter.lastSend = time.Now()
	c.rateLimiter.mutex.Unlock()

	c.logger.Debug().Str("message", data).Msg("send message")
	c.sendMutex.Lock()
	err := conn.WriteMessage(websocket.TextMessage, []byte(data))
	c.sendMutex.Unlock()
	if err != nil {
		c.logger.Error().Err(err).Str("message", data).Msg("failed to send message to websocket")
	}
}

// tickerLoop monitors connection health and triggers reconnection when needed.
// Exits cleanly when stopCh is closed.
func (c *BaseWsClient) tickerLoop() {
	c.logger.Info().Msg("tickerLoop started")
	for {
		c.stateMu.Lock()
		ticker := c.checkConnectionTicker
		c.stateMu.Unlock()

		select {
		case <-c.stopCh:
			c.logger.Info().Msg("tickerLoop stopped")
			return
		case <-ticker.C:
			c.stateMu.Lock()
			if c.reconnecting {
				c.stateMu.Unlock()
				continue
			}
			lastReceived := c.lastReceivedTime
			connStart := c.connectionStartTime
			timeout := c.reconnectionTimeout
			c.stateMu.Unlock()

			now := time.Now()
			elapsed := now.Sub(lastReceived)
			connectionAge := now.Sub(connStart)

			// 24-hour force disconnect per Bitget WebSocket spec
			if connectionAge > 24*time.Hour {
				c.logger.Info().Msg("24-hour limit reached, forcing WebSocket reconnection")
				go func() {
					if err := c.performReconnection(); err != nil {
						c.logger.Error().Err(err).Msg("Failed to perform 24-hour reconnection")
					}
				}()
				continue
			}

			// Message timeout
			if elapsed > timeout {
				c.logger.Warn().Dur("elapsed", elapsed).Msg("WebSocket reconnect due to timeout...")
				go func() {
					if err := c.performReconnection(); err != nil {
						c.logger.Error().Err(err).Msg("Failed to perform timeout reconnection")
					}
				}()
			}
		}
	}
}

// Reconnect manually triggers a WebSocket reconnection.
func (c *BaseWsClient) Reconnect() error {
	c.reconnectMutex.Lock()
	defer c.reconnectMutex.Unlock()

	c.stateMu.Lock()
	inProgress := c.reconnecting
	c.stateMu.Unlock()
	if inProgress {
		c.logger.Debug().Msg("Reconnection already in progress, skipping")
		return nil
	}

	c.logger.Info().Msg("Manual reconnection triggered")
	return c.performReconnection()
}

// performReconnection handles the actual reconnection logic with exponential backoff.
func (c *BaseWsClient) performReconnection() error {
	c.stateMu.Lock()
	c.reconnecting = true
	c.stateMu.Unlock()
	defer func() {
		c.stateMu.Lock()
		c.reconnecting = false
		c.stateMu.Unlock()
	}()

	c.disconnectWebSocket()

	c.stateMu.Lock()
	c.connected = false
	c.loginStatus = false
	c.reconnectAttempts = 0
	maxAttempts := c.maxReconnectAttempts
	c.stateMu.Unlock()

	for {
		// Check stopCh to abort reconnection during shutdown
		select {
		case <-c.stopCh:
			return fmt.Errorf("reconnection aborted: client closed")
		default:
		}

		c.stateMu.Lock()
		if maxAttempts > 0 && c.reconnectAttempts >= maxAttempts {
			attempts := c.reconnectAttempts
			c.stateMu.Unlock()
			c.logger.Error().
				Int("max_attempts", maxAttempts).
				Int("attempts", attempts).
				Msg("Maximum reconnection attempts reached, giving up")
			return fmt.Errorf("maximum reconnection attempts (%d) exceeded", maxAttempts)
		}
		c.reconnectAttempts++
		attempt := c.reconnectAttempts
		c.stateMu.Unlock()

		c.logger.Info().
			Int("attempt", attempt).
			Int("max_attempts", maxAttempts).
			Msg("Attempting to reconnect WebSocket")

		if err := c.attemptConnection(); err == nil {
			c.stateMu.Lock()
			c.reconnectAttempts = 0
			c.stateMu.Unlock()
			c.logger.Info().
				Int("attempts_used", attempt).
				Msg("WebSocket reconnection successful")
			return nil
		} else {
			c.logger.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("Reconnection attempt failed")
		}

		// Exponential backoff: wait 2^attempt seconds (max 30 seconds)
		backoff := time.Duration(1<<uint(attempt)) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}

		c.logger.Debug().
			Dur("backoff", backoff).
			Msg("Waiting before next reconnection attempt")

		select {
		case <-c.stopCh:
			return fmt.Errorf("reconnection aborted: client closed")
		case <-time.After(backoff):
		}
	}
}

// attemptConnection tries to establish a new WebSocket connection.
func (c *BaseWsClient) attemptConnection() error {
	c.logger.Debug().Str("url", c.url).Msg("Attempting WebSocket connection")

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	now := time.Now()
	c.stateMu.Lock()
	c.webSocketClient = conn
	c.connected = true
	c.connectionStartTime = now
	c.lastReceivedTime = now
	needLogin := c.needLogin
	hasCreds := c.storedLoginCreds != nil
	c.stateMu.Unlock()

	// Re-authenticate if needed
	if needLogin && hasCreds {
		c.logger.Info().Msg("Re-authenticating after reconnection")
		c.performLogin()
		// Brief wait for login response
		time.Sleep(1 * time.Second)
	}

	// Restore subscriptions
	if c.subscriptionCount() > 0 {
		c.logger.Info().Int("subscription_count", c.subscriptionCount()).Msg("Restoring subscriptions after reconnection")
		c.restoreSubscriptions()
	}

	return nil
}

// disconnectWebSocket closes the current connection and clears state.
func (c *BaseWsClient) disconnectWebSocket() {
	c.stateMu.Lock()
	conn := c.webSocketClient
	c.webSocketClient = nil
	c.connected = false
	c.stateMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			c.logger.Error().Interface("panic", r).Msg("Panic recovered during WebSocket disconnection")
		}
	}()

	if conn == nil {
		return
	}

	c.logger.Debug().Msg("WebSocket disconnecting...")
	if err := conn.Close(); err != nil {
		c.logger.Warn().Err(err).Msg("WebSocket disconnect error")
	} else {
		c.logger.Debug().Msg("WebSocket disconnected successfully")
	}
}

// ReadLoop reads messages from the WebSocket in a loop.
// Exits when stopCh is closed or the connection is closed.
func (c *BaseWsClient) ReadLoop() {
	for {
		select {
		case <-c.stopCh:
			c.logger.Info().Msg("ReadLoop stopped")
			return
		default:
		}

		c.stateMu.Lock()
		conn := c.webSocketClient
		c.stateMu.Unlock()

		if conn == nil {
			c.logger.Error().Msg("error on message read: no connection available")
			select {
			case <-c.stopCh:
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		_, buf, err := conn.ReadMessage()
		if err != nil {
			// Check if shutdown was requested — treat error as clean exit
			select {
			case <-c.stopCh:
				return
			default:
			}

			c.logger.Warn().Err(err).Str("msg", string(buf)).Msg("error on message read")

			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
				websocket.CloseProtocolError,
				websocket.CloseInternalServerErr,
				websocket.CloseServiceRestart,
			) {
				c.logger.Info().Msg("WebSocket closed, attempting reconnection")
				if rerr := c.performReconnection(); rerr != nil {
					c.logger.Error().Err(rerr).Msg("Failed to reconnect after close error")
				}
			}
			continue
		}

		now := time.Now()
		c.stateMu.Lock()
		c.lastReceivedTime = now
		c.stateMu.Unlock()

		message := string(buf)
		if message == "pong" {
			c.logger.Debug().Str("message", message).Msg("keep connected")
			continue
		}
		c.logger.Debug().Str("message", message).Msg("read message from websocket")

		jsonMap := make(map[string]interface{})
		if err := jsoniter.Unmarshal(buf, &jsonMap); err != nil {
			c.logger.Warn().Err(err).Msg("error on unmarshalling message")
			continue
		}

		if v, ok := jsonMap["code"]; ok {
			code, isNum := v.(float64)
			if !isNum || code != 0 {
				if c.errorListener != nil {
					c.errorListener(message)
				}
				continue
			}
		}

		if v, ok := jsonMap["event"]; ok && v == "login" {
			c.logger.Debug().Str("message", message).Msg("login")
			c.stateMu.Lock()
			c.loginStatus = true
			c.stateMu.Unlock()
			continue
		}

		if _, ok := jsonMap["data"]; ok {
			handler := c.getListener(jsonMap["arg"])
			if handler != nil {
				handler(message)
			}
			continue
		}
	}
}

// getSubscriptionArg returns the string representation of a subscription argument.
func getSubscriptionArg(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// getListener returns the handler registered for a subscription, or the default listener.
func (c *BaseWsClient) getListener(argJson any) OnReceive {
	mapData, ok := argJson.(map[string]any)
	if !ok {
		return c.listener
	}

	subscribeReq := SubscriptionArgs{
		ProductType: getSubscriptionArg(mapData["instType"]),
		Topic:       getSubscriptionArg(mapData["topic"]),
		Symbol:      getSubscriptionArg(mapData["symbol"]),
		Coin:        getSubscriptionArg(mapData["coin"]),
	}

	c.subsMu.RLock()
	handler, exists := c.subscriptions[subscribeReq]
	c.subsMu.RUnlock()

	if !exists {
		return c.listener
	}
	return handler
}

// IsConnected returns true if the WebSocket connection is established and active.
func (c *BaseWsClient) IsConnected() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.connected
}

// IsLoggedIn returns true if the WebSocket is authenticated (for private channels).
func (c *BaseWsClient) IsLoggedIn() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.loginStatus
}

// GetSubscriptionCount returns the number of active subscriptions.
func (c *BaseWsClient) GetSubscriptionCount() int {
	return c.subscriptionCount()
}

// subscriptionCount returns the current number of subscriptions (internal helper).
func (c *BaseWsClient) subscriptionCount() int {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()
	return len(c.subscriptions)
}

// setSubscription stores a handler for a subscription (internal helper, thread-safe).
func (c *BaseWsClient) setSubscription(args SubscriptionArgs, handler OnReceive) {
	c.subsMu.Lock()
	c.subscriptions[args] = handler
	c.subsMu.Unlock()
}

// deleteSubscription removes a subscription (internal helper, thread-safe).
func (c *BaseWsClient) deleteSubscription(args SubscriptionArgs) {
	c.subsMu.Lock()
	delete(c.subscriptions, args)
	c.subsMu.Unlock()
}

// restoreSubscriptions resubscribes to all previously active subscriptions after reconnection.
func (c *BaseWsClient) restoreSubscriptions() {
	c.logger.Info().Msg("Starting subscription restoration after reconnection")

	// Copy under read lock to avoid races while iterating.
	c.subsMu.RLock()
	subscriptionsToRestore := make([]SubscriptionArgs, 0, len(c.subscriptions))
	for args := range c.subscriptions {
		subscriptionsToRestore = append(subscriptionsToRestore, args)
	}
	c.subsMu.RUnlock()

	// Wait briefly for the connection to stabilize.
	time.Sleep(500 * time.Millisecond)

	c.stateMu.Lock()
	needLogin := c.needLogin
	hasCreds := c.storedLoginCreds != nil
	c.stateMu.Unlock()

	if needLogin && hasCreds {
		c.logger.Info().Msg("Re-authenticating private WebSocket after reconnection")
		c.performLogin()
		time.Sleep(500 * time.Millisecond)
	}

	restoredCount := 0
	for _, args := range subscriptionsToRestore {
		c.logger.Debug().
			Str("channel", args.Topic).
			Str("symbol", args.Symbol).
			Str("coin", args.Coin).
			Str("productType", args.ProductType).
			Msg("Restoring subscription")

		c.subscribe(args)
		restoredCount++

		time.Sleep(100 * time.Millisecond)
	}

	c.logger.Info().
		Int("restored_subscriptions", restoredCount).
		Int("total_subscriptions", c.subscriptionCount()).
		Msg("Subscription restoration completed")
}

// Close performs a graceful shutdown: stops goroutines, stops timers, closes the connection.
// Close is idempotent and safe to call multiple times.
func (c *BaseWsClient) Close() {
	c.closeOnce.Do(func() {
		// Signal goroutines (tickerLoop, ReadLoop, performReconnection) to exit.
		close(c.stopCh)

		c.stateMu.Lock()
		ticker := c.checkConnectionTicker
		cr := c.pingCron
		conn := c.webSocketClient
		connected := c.connected
		c.checkConnectionTicker = nil
		c.pingCron = nil
		c.webSocketClient = nil
		c.connected = false
		c.stateMu.Unlock()

		if ticker != nil {
			ticker.Stop()
		}
		if cr != nil {
			stopCtx := cr.Stop()
			// Wait briefly for in-flight cron jobs (ping) to finish.
			select {
			case <-stopCtx.Done():
			case <-time.After(2 * time.Second):
			}
		}

		if connected && conn != nil {
			cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close")
			// Best-effort graceful close notification.
			if err := conn.WriteMessage(websocket.CloseMessage, cm); err != nil {
				c.logger.Debug().Err(err).Msg("WebSocket close notification write error")
			}
			if err := conn.Close(); err != nil {
				c.logger.Debug().Err(err).Msg("WebSocket close error")
			}
		}
	})
}
