package monitor

import (
"encoding/json"
"os"
"path/filepath"
)

func GetCronJobs() ([]CronJob, error) {
var cronJobs []CronJob

home, err := os.UserHomeDir()
if err != nil {
return cronJobs, err
}

cronPath := filepath.Join(home, ".openclaw", "cron", "jobs.json")

data, err := os.ReadFile(cronPath)
if err != nil {
return cronJobs, nil // Return empty list if file doesn't exist
}

var jobsMap map[string]interface{}
if err := json.Unmarshal(data, &jobsMap); err != nil {
return cronJobs, nil
}

for id, jobData := range jobsMap {
jobMap, ok := jobData.(map[string]interface{})
if !ok {
continue
}

job := CronJob{
Name: id,
}

if name, ok := jobMap["name"].(string); ok {
job.Name = name
}
if schedule, ok := jobMap["schedule"].(string); ok {
job.Schedule = schedule
}
if status, ok := jobMap["status"].(string); ok {
job.Status = status
}

cronJobs = append(cronJobs, job)
}

return cronJobs, nil
}
