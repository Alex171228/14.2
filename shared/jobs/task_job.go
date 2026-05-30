package jobs

import (
	"fmt"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	DefaultQueueName    = "task_jobs"
	DefaultDLQName      = "task_jobs_dlq"
	DefaultJobName      = "process_task"
	DefaultMaxAttempts  = 3
	DefaultWorkerDelayS = 2
)

type TaskJob struct {
	Job       string `json:"job"`
	TaskID    string `json:"task_id"`
	Attempt   int    `json:"attempt"`
	MessageID string `json:"message_id"`
}

func DeclareQueues(ch *amqp091.Channel, queueName, dlqQueueName string) error {
	if _, err := ch.QueueDeclare(
		dlqQueueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}

	args := amqp091.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlqQueueName,
	}

	if _, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	return nil
}
