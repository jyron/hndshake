-- Migration: 002_age_to_integer
-- Description: Change age_range (bucket) to exact age (integer)
-- This migration converts the age bucket system to an exact age field

-- Add new age column
ALTER TABLE posts ADD COLUMN IF NOT EXISTS age INTEGER;

-- Migrate data from age_range to age (use middle of range as best approximation)
UPDATE posts SET age = CASE 
    WHEN age_range = 'under-18' THEN 16
    WHEN age_range = '18-24' THEN 21
    WHEN age_range = '25-34' THEN 30
    WHEN age_range = '35-44' THEN 40
    WHEN age_range = '45-54' THEN 50
    WHEN age_range = '55-64' THEN 60
    WHEN age_range = '65+' THEN 70
    ELSE NULL
END
WHERE age IS NULL;

-- Make age NOT NULL after migration (only if there's data)
-- Note: We keep this optional in case you want to run it separately
-- ALTER TABLE posts ALTER COLUMN age SET NOT NULL;

-- Drop the old age_range column
ALTER TABLE posts DROP COLUMN IF EXISTS age_range;

-- Add check constraint for reasonable age range
ALTER TABLE posts ADD CONSTRAINT check_age_range CHECK (age >= 1 AND age <= 120);
