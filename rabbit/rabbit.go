package rabbit

import (
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type channel interface {
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	ExchangeDeclare(name, typ string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
}

type connection interface {
	Channel() (*amqp.Channel, error)
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Close() error
}

// Client provides simple abstration over streadway/amqp package
type Client struct {
	conn        connection
	serviceName string
	closeCh     chan error
}

type Config struct {
	ServiceName string `validate:"nonzero"`
	Endpoint    string `validate:"nonzero"`
}

type ExchangeOptions struct {
	Name    string `validate:"nonzero"`
	Type    string `validate:"nonzero"`
	Durable bool
}

func New(config Config) (*Client, error) {
	var conn connection
	conn, err := amqp.Dial(config.Endpoint)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to dial")
	}

	amqpErrCh := conn.NotifyClose(make(chan *amqp.Error))

	client := Client{
		conn:        conn,
		serviceName: config.ServiceName,
		closeCh:     make(chan error, 1),
	}

	go func() {
		closeErr := <-amqpErrCh
		if closeErr != nil {
			client.closeCh <- closeErr
		}
		close(client.closeCh)
	}()

	return &client, nil
}

func (c *Client) Done() <-chan error {
	return c.closeCh
}

func MustNew(config Config) *Client {
	c, err := New(config)
	if err != nil {
		panic(err)
	}

	return c
}

func (c *Client) NewExchange(options ExchangeOptions) (*Exchange, error) {
	return newExchange(c.conn, options, c.serviceName)
}

func (c *Client) MustNewExchange(options ExchangeOptions) *Exchange {
	ex, err := c.NewExchange(options)
	if err != nil {
		panic(err)
	}

	return ex
}
