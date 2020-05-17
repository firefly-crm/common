package rabbit

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"

	"github.com/firefly-crm/common/logger"
)

type Queue struct {
	options     QueueOptions
	q           amqp.Queue
	conn        connection
	serviceName string
}

type QueueOptions struct {
	Name           string
	RoutingKey     string
	WorkerPoolSize int
	Qos            int
	GlobalQos      bool
}

type UnmarshalError error

type workerFunc func(ctx context.Context, d amqp.Delivery) error

func (q *Queue) Consume(ctx context.Context, consumer interface{}) error {
	log := logger.FromContext(ctx).WithField("queue", q.options.Name)

	ch, err := q.conn.Channel()
	if err != nil {
		return errors.WithMessage(err, "failed to open a channel for consumer")
	}
	defer func() {
		err := ch.Close()
		if err != nil {
			log.Warnf("failed to close channel: %w", err)
		}
	}()

	if q.options.Qos != 0 {
		err := ch.Qos(q.options.Qos, 0, q.options.GlobalQos)
		if err != nil {
			return errors.WithMessage(err, "failed to set qos")
		}
	}

	deliveryCh, err := ch.Consume(
		q.q.Name,      // queue
		q.serviceName, // consumer
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)

	if err != nil {
		return errors.WithMessage(err, "failed to consume messages")
	}

	worker, err := toWorker(consumer)
	if err != nil {
		return errors.WithMessage(err, "failed to create a consumer")
	}

	workersLimitChan := make(chan struct{}, q.options.WorkerPoolSize)

	for i := 0; i < q.options.WorkerPoolSize; i++ {
		workersLimitChan <- struct{}{}
	}

	closeChan := q.conn.NotifyClose(make(chan *amqp.Error))

	log.Debugf("started to consume events")
	defer log.Debugf("finished consuming events")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case connErr := <-closeChan:
			return errors.WithMessage(connErr, "amqp connection was closed")
		case d, open := <-deliveryCh:
			if !open {
				return errors.Errorf("delivery channel is closed for the queue: %s(%s)", q.options.Name, q.q.Name)
			}

			<-workersLimitChan

			go func() {
				workerCtx := logger.ToContext(context.Background(), log)

				if err := worker(workerCtx, d); err != nil {
					log.Errorf("consumer returned an error: %v", err)

					_, isUnmarshalErr := errors.Cause(err).(UnmarshalError)

					//do not requeue messages that we can't unmarshal put them to DLX instead
					if err = d.Acknowledger.Reject(d.DeliveryTag, !isUnmarshalErr); err != nil {
						log.Errorf("failed to send nack: %v", err)
					}
				} else {
					if err := d.Acknowledger.Ack(d.DeliveryTag, false); err != nil {
						log.Errorf("failed to send ack: %v", err)
					}
				}
				workersLimitChan <- struct{}{}
			}()
		}
	}
}

// toWorker checks that given func can be used as
// func(context.Context, proto.Message) error
// and converts it to workerFunc
func toWorker(f interface{}) (workerFunc, error) {
	hType := reflect.TypeOf(f)
	hValue := reflect.ValueOf(f)

	if hValue.Kind() != reflect.Func {
		return nil, errors.Errorf("expected a function, got %s", hValue.Kind())
	}

	numIn := hType.NumIn()
	if numIn != 2 {
		return nil, errors.Errorf("unexpected number of arguments: %d", numIn)
	}

	firstInputParam := hType.In(0)
	if !firstInputParam.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, errors.New("first argument must implement the context.Context interface")
	}

	inType := hType.In(1)
	numOut := hType.NumOut()

	if !inType.Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
		return nil, errors.New("second argument must implement the proto.Message interface")
	}

	if numOut != 1 {
		return nil, errors.Errorf("unexpected number of results: %d", numOut)
	}

	result := hType.Out(0)
	if !result.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, errors.New("result does not implement the error interface")
	}

	return func(ctx context.Context, d amqp.Delivery) error {
		in := reflect.New(inType.Elem()).Interface().(proto.Message)

		if err := proto.Unmarshal(d.Body, in); err != nil {
			return errors.WithMessage(UnmarshalError(err), "failed to unmarshal delivery body")
		}

		log := logger.FromContext(ctx)

		//TODO: Add tracing
		//spCtx, _ := amqptracer.Extract(d.Headers)
		//span := opentracing.StartSpan(
		//	"Consume",
		//	opentracing.FollowsFrom(spCtx),
		//)
		//defer span.Finish()

		//ctx = opentracing.ContextWithSpan(ctx, span)
		//log := logger.WithTraceId(logger.FromContext(ctx), span)
		//ctx = logger.ToContext(ctx, log)

		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(in),
		}

		log.Debugf("processing message")
		result := hValue.Call(args)

		errorInterface := result[0].Interface()
		if errorInterface != nil {
			return errorInterface.(error)
		}

		return nil
	}, nil
}
