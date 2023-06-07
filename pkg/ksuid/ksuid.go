package ksuid

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/keybase/saltpack/encoding/basex"
)

type KSUID [20]byte

const Epoch = 1_400_000_000

var Zero = KSUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

var ErrInvalidFormat = errors.New("invalid format")

func Generate() KSUID {
	return GenerateWithTime(time.Now())
}

func GenerateWithTime(t time.Time) KSUID {
	timestamp := t.Unix()

	if timestamp < Epoch {
		panic(fmt.Sprintf("timestamp %d too small (min: %d)",
			timestamp, Epoch))
	}

	if timestamp > math.MaxUint32 {
		panic(fmt.Sprintf("timestamp %d too large (max: %d)",
			timestamp, math.MaxUint32))
	}

	return GenerateWithTimestamp(uint32(timestamp - Epoch))
}

func GenerateWithTimestamp(timestamp uint32) KSUID {
	var id KSUID

	binary.BigEndian.PutUint32(id[0:4], timestamp)

	if _, err := rand.Read(id[4:20]); err != nil {
		panic(fmt.Sprintf("cannot read random data: %v", err))
	}

	return id
}

func (id *KSUID) Parse(s string) error {
	if len(s) != 27 {
		return ErrInvalidFormat
	}

	data, err := Base62Decode(s)
	if err != nil {
		return ErrInvalidFormat
	}

	copy(id[0:20], data)

	return nil
}

func (id *KSUID) ValueOrZero() KSUID {
	if id == nil {
		return Zero
	}

	return *id
}

func (id KSUID) String() string {
	return Base62Encode(id[:])
}

func (id KSUID) GoString() string {
	return id.String()
}

func (id KSUID) Timestamp() uint32 {
	return binary.BigEndian.Uint32(id[0:4])
}

func (id KSUID) Time() time.Time {
	return time.Unix(int64(Epoch+id.Timestamp()), 0).UTC()
}

func (id KSUID) IsZero() bool {
	return id == Zero
}

func (id KSUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *KSUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid value")
	}

	return id.Parse(s)
}

// sql.Scanner interface
func (id *KSUID) Scan(src interface{}) error {
	if src == nil {
		*id = Zero
		return nil
	}

	switch v := src.(type) {
	case string:
		return id.Parse(v)

	default:
		return fmt.Errorf("invalid value of type %T", v)
	}
}

// sql/driver.Valuer interface
func (id KSUID) Value() (driver.Value, error) {
	return id.String(), nil
}

func Base62Encode(data []byte) string {
	return basex.Base62StdEncodingStrict.EncodeToString(data)
}

func Base62Decode(s string) ([]byte, error) {
	return basex.Base62StdEncodingStrict.DecodeString(s)
}
