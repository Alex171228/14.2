package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"go.uber.org/zap"

	"pz1.2/services/worker/internal/consumer"
	"pz1.2/shared/logger"
)

func main() {
	log := logger.New("worker")
	defer log.Sync()

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	queueName := os.Getenv("JOB_QUEUE_NAME")
	if queueName == "" {
		queueName = "task_jobs"
	}

	dlqQueueName := os.Getenv("JOB_DLQ_NAME")
	if dlqQueueName == "" {
		dlqQueueName = "task_jobs_dlq"
	}

	prefetch := 1
	if v := os.Getenv("WORKER_PREFETCH"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			prefetch = parsed
		}
	}

	maxAttempts := 3
	if v := os.Getenv("MAX_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxAttempts = parsed
		}
	}

	consumer, err := consumer.New(rabbitURL, queueName, dlqQueueName, prefetch, maxAttempts, log)
	if err != nil {
		log.Fatal("failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := consumer.Run(ctx); err != nil {
		log.Fatal("worker stopped with error", zap.Error(err))
	}
}
