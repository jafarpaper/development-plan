package activitylog

import "time"

type ActivityLog struct {
	ID               string     `gorm:"column:id;primaryKey" json:"id"`
	ActivityName     string     `gorm:"column:activity_name" json:"activity_name"`
	CompanyID        string     `gorm:"column:company_id" json:"company_id"`
	ObjectName       string     `gorm:"column:object_name" json:"object_name"`
	ObjectID         string     `gorm:"column:object_id" json:"object_id"`
	Changes          string     `gorm:"column:changes;type:longtext" json:"changes"`
	FormattedMessage string     `gorm:"column:formatted_message;type:text" json:"formatted_message"`
	ActorID          string     `gorm:"column:actor_id" json:"actor_id"`
	ActorName        string     `gorm:"column:actor_name" json:"actor_name"`
	ActorEmail       string     `gorm:"column:actor_email" json:"actor_email"`
	CreatedAt        *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (*ActivityLog) TableName() string {
	return "activity_log"
}
