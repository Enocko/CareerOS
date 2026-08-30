-- Students choose their own university; do not pre-fill a school name.
ALTER TABLE student_profiles ALTER COLUMN university DROP DEFAULT;
