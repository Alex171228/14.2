package amqp

import (
	"context"
	"encoding/json"
	"fmt"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"pz1.2/shared/jobs"
)

type JobPublisher struct {
	*Publisher
	dlqQueueName string
}

func NewJobPublisher(rabbitURL, queueName, dlqQueueName string, log *zap.Logger) *JobPublisher {
	return &JobPublisher{
		Publisher: &Publisher{
			rabbitURL: rabbitURL,
			queueName: queueName,
			log:       log.With(zap.String("component", "task_job_publisher")),
		},
		dlqQueueName: dlqQueueName,
	}
}

func (p *JobPublisher) PublishJob(ctx context.Context, job jobs.TaskJob) error {
	conn, err := p.getConnection()
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if err := jobs.DeclareQueues(ch, p.queueName, p.dlqQueueName); err != nil {
		return err
	}

	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	if err := ch.PublishWithContext(
		ctx,
		"",
		p.queueName,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			MessageId:    job.MessageID,
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish job: %w", err)
	}

	p.log.Info(
		"task job published",
		zap.String("job", job.Job),
		zap.String("task_id", job.TaskID),
		zap.Int("attempt", job.Attempt),
		zap.String("message_id", job.MessageID),
	)
	return nil
}
