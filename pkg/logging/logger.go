package logging

import (
	"time"

    "go.uber.org/zap"
)

type Logger struct {
    logger *zap.Logger
}

func NewLogger() *Logger {
    config := zap.NewProductionConfig()
    logger, _ := config.Build()
    return &Logger{logger: logger}
}

func (l *Logger) LogTaskExecution(task *TaskInstance, duration time.Duration, err error) {
    l.logger.Info("task execution",
        zap.String("taskId", task.ID),
        zap.String("definitionId", task.DefinitionID),
        zap.Duration("duration", duration),
        zap.Error(err),
    )
}