package options

import "tracker-server/internal/storage"

// WithCheckBusinessDay sets the checkBusinessDay option
func WithCheckBusinessDay(check bool) storage.TaskRecordOption {
	return func(o *storage.TaskRecordOptions) {
		o.CheckBusinessDay = check
	}
}
