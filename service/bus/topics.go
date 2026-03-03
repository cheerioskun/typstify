package bus

const (
	TopicSettingsUpdated      = "settings.updated"
	TopicStatusbarNotifyEvent = "statusbar.notification"
	TopicProjectSwitched      = "project.switched"
	// request to create a new project.
	TopicProjectCreate = "project.create"
)

var allTopics = []string{
	TopicSettingsUpdated,
	TopicStatusbarNotifyEvent,
	TopicProjectSwitched,
	TopicProjectCreate,
}

