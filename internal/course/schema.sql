CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    capacity INTEGER NOT NULL CHECK(capacity>0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);