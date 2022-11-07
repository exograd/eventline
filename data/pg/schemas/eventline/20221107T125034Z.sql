CREATE FUNCTION truncate_string(IN p_string VARCHAR,
                                IN p_max_size INTEGER,
                                IN p_truncation_string VARCHAR)
RETURNS VARCHAR
AS
$$
DECLARE
BEGIN
  IF octet_length(p_string) > p_max_size THEN
    RETURN substring(p_string for p_max_size) || p_truncation_string;
  ELSE
    RETURN p_string;
  END IF;
END
$$
LANGUAGE PLPGSQL
IMMUTABLE
STRICT
PARALLEL SAFE;
