package rabbit

import (
	"context"
	"github.com/firefly-crm/common/rabbit/routes"

	"github.com/firefly-crm/common/logger"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing-contrib/go-amqp/amqptracer"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type Exchange struct {
	ch          channel
	conn        connection
	options     ExchangeOptions
	serviceName string
}

func newExchange(conn connection, options ExchangeOptions, serviceName string) (*Exchange, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to open a channel")
	}

	if err := ch.ExchangeDeclare(options.Name, options.Type, options.Durable, false, false, false, nil); err != nil {
		return nil, errors.WithMessage(err, "failed to declare exchange")
	}

	return &Exchange{
		ch:          ch, //this channel is used only for publishing to the exchange
		options:     options,
		conn:        conn,
		serviceName: serviceName,
	}, nil
}

func (e *Exchange) NewQueue(options QueueOptions) (*Queue, error) {
	ch, err := e.conn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to open a channel")
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		options.Name,
		e.options.Durable,
		false, // delete when usused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to declare a queue")
	}

	err = ch.QueueBind(
		options.Name,
		options.RoutingKey,
		e.options.Name,
		false, //nowait
		nil,   //arguments
	)

	if err != nil {
		return nil, errors.WithMessage(err, "failed to bind a queue")
	}

	return &Queue{options: options, q: q, conn: e.conn, serviceName: e.serviceName}, nil
}

func (e *Exchange) MustNewQueue(options QueueOptions) *Queue {
	q, err := e.NewQueue(options)
	if err != nil {
		panic(err)
	}
	return q
}

func (e *Exchange) Queue(id routes.Route) *Queue {
	route, err := routes.QueueByID(id)
	if err != nil {
		panic(errors.WithMessage(err, "failed to get a queue"))
	}
	op := QueueOptions{
		Name:           route.Name(),
		RoutingKey:     route.Route(),
		WorkerPoolSize: route.WorkerPoolSize(),
		Qos:            route.Qos(),
		GlobalQos:      route.GlobalQos(),
	}
	q, err := e.NewQueue(op)
	if err != nil {
		panic(err)
	}
	return q
}

func (e *Exchange) QueueWildcard(id routes.Route, param string) *Queue {
	route, err := routes.QueueByID(id)
	if err != nil {
		panic(errors.WithMessage(err, "failed to get a queue"))
	}
	op := QueueOptions{
		Name:           route.NameByParam(param),
		RoutingKey:     route.RouteWildcard(),
		WorkerPoolSize: route.WorkerPoolSize(),
		Qos:            route.Qos(),
		GlobalQos:      route.GlobalQos(),
	}
	q, err := e.NewQueue(op)
	if err != nil {
		panic(err)
	}
	return q
}

func (e *Exchange) QueueByParam(id routes.Route, param string) *Queue {
	route, err := routes.QueueByID(id)
	if err != nil {
		panic(errors.WithMessage(err, "failed to get a queue"))
	}
	op := QueueOptions{
		Name:           route.NameByParam(param),
		RoutingKey:     route.RouteByParam(param),
		WorkerPoolSize: route.WorkerPoolSize(),
		Qos:            route.Qos(),
		GlobalQos:      route.GlobalQos(),
	}
	q, err := e.NewQueue(op)
	if err != nil {
		panic(err)
	}
	return q
}

func (e *Exchange) Publish(ctx context.Context, routingKey string, m proto.Message) error {
	//if validatable, ok := m.(interface{ Validate() error }); ok {
	//	if err := validatable.Validate(); err != nil {
	//		return errors.WithMessagef(err, "failed to publish invalid message: %#v", m)
	//	}
	//}

	b, err := proto.Marshal(m)
	if err != nil {
		return errors.WithMessage(err, "failed to marshal message")
	}

	msg := amqp.Publishing{
		ContentType: "text/plain",
		Body:        b,
		Headers:     make(amqp.Table),
	}

	sp := opentracing.SpanFromContext(ctx)
	defer sp.Finish()

	if err := amqptracer.Inject(sp, msg.Headers); err != nil {
		return errors.WithMessage(err, "failed to inject span information to headers")
	}

	if e.options.Durable {
		msg.DeliveryMode = 2 //Persistent
	}

	logger.FromContext(ctx).
		WithField("exchange", e.options.Name).
		WithField("routingKey", routingKey).
		Debugf("publishing event to exchange")

	return e.ch.Publish(e.options.Name, routingKey, false, false, msg)
}
