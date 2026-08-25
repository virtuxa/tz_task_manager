package stats

type StatusCount struct {
	Status string
	Count  int64
}

type AssigneeStat struct {
	UserID           int64
	Name             string
	ClosedTasksCount int64
}

type Stats struct {
	TeamID                      int64
	TasksByStatus               []StatusCount
	TopAssignees                []AssigneeStat
	AverageCloseDurationSeconds float64
	CommentsCount               int64
}
