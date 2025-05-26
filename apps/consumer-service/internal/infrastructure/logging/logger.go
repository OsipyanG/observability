package logging

import (
	"consumer-service/internal/domain"

	"github.com/sirupsen/logrus"
)

// LogrusAdapter адаптер для logrus, реализующий интерфейс domain.Logger
type LogrusAdapter struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusAdapter создает новый адаптер для logrus
func NewLogrusAdapter(logger *logrus.Logger) domain.Logger {
	return &LogrusAdapter{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// Debug логирует debug сообщение
func (l *LogrusAdapter) Debug(msg string, fields ...interface{}) {
	l.entry.WithFields(l.parseFields(fields...)).Debug(msg)
}

// Info логирует info сообщение
func (l *LogrusAdapter) Info(msg string, fields ...interface{}) {
	l.entry.WithFields(l.parseFields(fields...)).Info(msg)
}

// Warn логирует warning сообщение
func (l *LogrusAdapter) Warn(msg string, fields ...interface{}) {
	l.entry.WithFields(l.parseFields(fields...)).Warn(msg)
}

// Error логирует error сообщение
func (l *LogrusAdapter) Error(msg string, fields ...interface{}) {
	l.entry.WithFields(l.parseFields(fields...)).Error(msg)
}

// WithField добавляет поле к логгеру
func (l *LogrusAdapter) WithField(key string, value interface{}) domain.Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields добавляет несколько полей к логгеру
func (l *LogrusAdapter) WithFields(fields map[string]interface{}) domain.Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithFields(logrus.Fields(fields)),
	}
}

// parseFields парсит поля из variadic аргументов
func (l *LogrusAdapter) parseFields(fields ...interface{}) logrus.Fields {
	logrusFields := make(logrus.Fields)

	// Парсим поля в формате key, value, key, value...
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			logrusFields[key] = fields[i+1]
		}
	}

	return logrusFields
}
