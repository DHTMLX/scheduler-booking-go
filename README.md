# Scheduler-Booking Demo Backend

## How to use

```
go build
./scheduler-booking
```

# API

### GET /units

Returns all neccessary information to build booking dataset. Using `slots + usedslots` approach

#### Response example

```js
{
  "id": 1,
  "title": "Dr. Conrad Hubbard",
  "category": "Psychiatrist",
  "subtitle": "2 years of experience",
  "details": "Desert Springs Hospital (Schroeders Avenue 90, Fannett, Ethiopia)",
  "preview": "",
  "price": 120,
  "slots": [
    {
      "from": "9:00",
      "to": "14:00",
      "size": 45,
      "gap": 5,
      "days": [1, 3, 5] // reccuring events
    },
    {
      "from": "15:30",
      "to": "20:00",
      "size": 45,
      "gap": 5,
      "dates": [1695254400000] // Thu Sep 21 2023
    },
    ...
  ],
  "usedSlots": [
    1695367800000, // Fri Sep 22 2023 10:30:00 AM
    ...
  ]
}
```

### GET /doctors

Returns a list of doctors (without images)

#### Response example

```js
[
  {
    "id": 1,
    "name": "Dr. Conrad Hubbard",
    "subtitle": "2 years of experience",
    "details": "Desert Springs Hospital (Schroeders Avenue 90, Fannett, Ethiopia)",
    "category": "Psychiatrist",
    "price": 45,
    "gap": 20,
    "slot_size": 20
  },
  ...
]
```

### GET /doctors/worktime

Returns a list of doctor's schedule (excluding expired dates).
You can show this data on Doctors view in Booking-Scheduler Demo

#### Response example

```js
[
  {
    "id": 1,
    "doctor_id": 1,
    "start_date": "2024-10-28 09:00:00",
    "duration": 28800, // in seconds (8 hours)
    "end_date": "2026-10-28 00:00:00",
    "rrule": "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR" // reccuring events
  },
  {
    "id": 2,
    "doctor_id": 1,
    "start_date": "2024-10-29 18:00:00",
    "end_date": "2024-10-29 22:00:00"
  },
  {
    "id": 3,
    "doctor_id": 1,
    "start_date": "2024-10-30 18:00:00",
    "end_date": "2024-10-30 22:00:00"
  },
  // extension
  {
    "id": 4,
    "doctor_id": 1,
    "start_date": "2024-10-31 09:00:00",
    "end_date": "2024-10-31 17:00:00",
    "recurring_event_id": 1,
    "original": "2024-10-30 09:00:00"
  },
  // removed extension
  {
    "id": 5,
    "doctor_id": 1,
    "start_date": "2024-11-2 09:00:00",
    "end_date": "2024-11-2 17:00:00",
    "recurring_event_id": 1,
    "original": "2024-11-01 09:00:00",
    "deleted": true
  }
  ...
]
```

### POST /doctors/worktime

Creates a new doctor's schedule with **concrete date** (Doctors view)

#### Body

```js
{
  "doctor_id": 1,
  "start_date": "2024-10-31 10:30",
  "end_date": "2024-10-31 14:30"
}
```

Creates a new doctor's schedule with **recurring days** (Doctors view)

#### Body

```js
{
  "doctor_id": 1,
  "start_date": "2024-10-28 10:30",
  "duration":	14400, // in seconds (4 hours)
  "end_date": "2026-10-28 00:00:00",
  "rrule":	"FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR"
}
```

### Response example

Returns an ID and action of created schedule (Doctors view)

```js
{
  "tid": 10,
  "action": "inserted" // or "deleted" if removed event was created for recurring event
}
```

### PUT /doctors/worktime/{id}

Updates doctor's schedule

#### Body

```js
{
  "doctor_id": 1,
  "start_date": "2024-10-31 12:20",
  "end_date": "2024-10-31 16:55"
}
```

Updates **recurring** doctor's schedule

#### Body

```js
{
  "doctor_id": 1,
  "start_date": "2024-10-31 10:30",
  "duration":	14400, // in seconds (4 hours)
  "end_date": "2026-10-28 00:00:00",
  "rrule":	"FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,WE,FR"
}
```

### Response example

Returns an ID and action of updated schedule (Doctors view)

```js
{
  "tid": 10,
  "action": "updated"
}
```

#### URL Params:

- id [required] - ID of the schedule to be updated

### DELETE /doctors/worktime/{id}

Deletes doctor's schedule (Doctors view)

### Response example

Returns an ID and action of updated schedule (Doctors view)

```js
{
  "action": "deleted"
}
```

#### URL Params:

- id [required] - ID of the schedule to be deleted

### GET /doctors/reservations

Returns all occupied slots (Clients view)

#### Response example

```js
[
    {
        "id": 1,
        "doctor_id": 2,
        "date": 1730289600000,
        "client_name": "Alan",
        "client_email": "alan@gmail.com",
        "client_details": ""
    },
    {
        "id": 2,
        "doctor_id": 3,
        "date": 1730356200000,
        "client_name": "Viron",
        "client_email": "viron@hr.com",
        "client_details": ""
    }
    ...
]
```

### POST /doctors/reservations

Creates reservation (Booking view)

#### Body

```js
{
  "doctor": 2,
  "date": 1730289600000,
  "form": {
    "name": "Alan",
    "email": "alan@gmail.com",
    "details": ""
  }
}
```

# Features

### Booking schedules

If the schedule encompasses midnight and there is enough time for a time slot after it, then the schedule is divided into two parts

### Used slots

Booking processes only matches exact used slots for the doctor. If the booked slot does not match any of the slots, the two closest relevant slots will be booked instead

# Config

```yaml
db:
  path: db.sqlite    # path to the database
  resetonstart: true # reset data on server restart
server:
  url: "http://localhost:3000"
  port: ":3000"
  cors:
    - "*"
  resetFrequence: 120 # every 2 hours restart data (value in minutes)
```
