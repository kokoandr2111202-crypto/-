create table exercises (
  id serial primary key,
  name text not null unique,
  created_at timestamptz not null default now()
);

create table workouts (
  id serial primary key,
  started_at timestamptz not null,
  ended_at timestamptz,
  created_at timestamptz not null default now()
);

create table workout_sets (
  id serial primary key,
  workout_id integer not null references workouts(id) on delete cascade,
  exercise_id integer not null references exercises(id),
  weight numeric not null,
  reps integer not null,
  created_at timestamptz not null default now()
);

create index idx_sets_workout on workout_sets(workout_id);
create index idx_sets_exercise on workout_sets(exercise_id);
