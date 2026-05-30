package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"pz1.2/services/worker/internal/store"
	"pz1.2/shared/jobs"
)

type Consumer struct {
	conn         *amqp091.Connection
	queueName    string
	dlqQueueName string
	prefetch     int
	maxAttempts  int
	processed    *store.ProcessedStore
	log          *zap.Logger
}

func New(rabbitURL, queueName, dlqQueueName string, prefetch, maxAttempts int, log *zap.Logger) (*Consumer, error) {
	if prefetch <= 0 {
		prefetch = 1
	}
	if maxAttempts <= 0 {
		maxAttempts = jobs.DefaultMaxAttempts
	}

	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	return &Consumer{
		conn:         conn,
		queueName:    queueName,
		dlqQueueName: dlqQueueName,
		prefetch:     prefetch,
		maxAttempts:  maxAttempts,
		processed:    store.NewProcessedStore(),
		log:          log.With(zap.String("component", "worker_consumer")),
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if err := jobs.DeclareQueues(ch, c.queueName, c.dlqQueueName); err != nil {
		return err
	}

	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("configure qos: %w", err)
	}

	msgs, err := ch.Consume(
		c.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	c.log.Info(
		"worker started",
		zap.String("queue", c.queueName),
		zap.String("dlq_queue", c.dlqQueueName),
		zap.Int("prefetch", c.prefetch),
		zap.Int("max_attempts", c.maxAttempts),
	)

	for {
		select {
		case <-ctx.Done():
			c.log.Info("worker stopping")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}

			var job jobs.TaskJob
			if err := json.Unmarshal(d.Body, &job); err != nil {
				c.log.Warn("bad message", zap.Error(err))
				if nackErr := d.Nack(false, false); nackErr != nil {
					c.log.Warn("nack failed", zap.Error(nackErr))
				}
				continue
			}

			if job.Attempt <= 0 {
				job.Attempt = 1
			}

			if job.MessageID == "" {
				c.log.Warn("job message_id is empty")
				if nackErr := d.Nack(false, false); nackErr != nil {
					c.log.Warn("nack failed", zap.Error(nackErr))
				}
				continue
			}

			if c.processed.Exists(job.MessageID) {
				c.log.Info("duplicate job skipped", zap.String("message_id", job.MessageID), zap.String("task_id", job.TaskID))
				if err := d.Ack(false); err != nil {
					c.log.Warn("ack failed", zap.Error(err))
				}
				continue
			}

			c.log.Info(
				"task job received",
				zap.String("job", job.Job),
				zap.String("task_id", job.TaskID),
				zap.Int("attempt", job.Attempt),
				zap.String("message_id", job.MessageID),
			)

			if err := c.processTask(job); err != nil {
				c.log.Warn(
					"task job failed",
					zap.String("task_id", job.TaskID),
					zap.Int("attempt", job.Attempt),
					zap.String("message_id", job.MessageID),
					zap.Error(err),
				)

				job.Attempt++
				targetQueue := c.dlqQueueName
				logMsg := "job published to dlq"
				if job.Attempt <= c.maxAttempts {
					targetQueue = c.queueName
					logMsg = "job scheduled for retry"
				}

				if err := c.publishJob(ch, targetQueue, job); err != nil {
					c.log.Warn("republish failed", zap.String("queue", targetQueue), zap.Error(err))
					if nackErr := d.Nack(false, true); nackErr != nil {
						c.log.Warn("nack failed", zap.Error(nackErr))
					}
					continue
				}

				c.log.Info(logMsg, zap.String("queue", targetQueue), zap.String("task_id", job.TaskID), zap.Int("attempt", job.Attempt), zap.String("message_id", job.MessageID))
				if err := d.Ack(false); err != nil {
					c.log.Warn("ack failed", zap.Error(err))
				}
				continue
			}

			c.processed.MarkDone(job.MessageID)
			c.log.Info("task job processed", zap.String("task_id", job.TaskID), zap.String("message_id", job.MessageID))

			if err := d.Ack(false); err != nil {
				c.log.Warn("ack failed", zap.Error(err))
			}
		}
	}
}

func (c *Consumer) processTask(job jobs.TaskJob) error {
	time.Sleep(jobs.DefaultWorkerDelayS * time.Second)
	if job.TaskID == "t_fail" {
		return fmt.Errorf("simulated processing error")
	}
	return nil
}

func (c *Consumer) publishJob(ch *amqp091.Channel, queueName string, job jobs.TaskJob) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	return ch.PublishWithContext(
		context.Background(),
		"",
		queueName,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			MessageId:    job.MessageID,
			Body:         body,
		},
	)
}

func (c *Consumer) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
