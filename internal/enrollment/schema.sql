CREATE TABLE enrollments (
    member_id UUID
                         REFERENCES members(id)
                         ON DELETE CASCADE,
    course_id UUID
                         REFERENCES courses(id)
                         ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'enrolled' CHECK(status IN('enrolled', 'completed', 'cancelled')),
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (member_id, course_id)
);