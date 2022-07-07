package eventline

import (
	"bytes"
	"encoding/json"
)

type JobExecutionInput struct {
	Parameters    map[string]interface{} `json:"-"`
	RawParameters json.RawMessage        `json:"parameters"`
}

func (pi *JobExecutionInput) MarshalJSON() ([]byte, error) {
	type JobExecutionInput2 JobExecutionInput
	i := JobExecutionInput2(*pi)

	if i.RawParameters == nil {
		data, err := json.Marshal(i.Parameters)
		if err != nil {
			return nil, err
		}

		i.RawParameters = data
	}

	return json.Marshal(i)
}

func (pi *JobExecutionInput) UnmarshalJSON(data []byte) error {
	type JobExecutionInput2 JobExecutionInput
	i := JobExecutionInput2(*pi)

	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(i.RawParameters))
	decoder.UseNumber() // necessary to correctly decode parameters

	if err := decoder.Decode(&i.Parameters); err != nil {
		return err
	}

	*pi = JobExecutionInput(i)
	return nil
}
