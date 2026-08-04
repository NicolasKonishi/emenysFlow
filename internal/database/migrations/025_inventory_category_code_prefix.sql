ALTER TABLE inventory_categories ADD COLUMN internal_code_prefix TEXT NOT NULL DEFAULT '';

UPDATE inventory_categories
SET internal_code_prefix = CASE id
    WHEN 1 THEN 'COM'
    WHEN 2 THEN 'REC'
    WHEN 3 THEN 'CUB'
    WHEN 4 THEN 'EQP'
    WHEN 5 THEN 'LOU'
    WHEN 6 THEN 'BEB'
    WHEN 7 THEN 'DES'
    WHEN 8 THEN 'GAR'
    WHEN 9 THEN 'CAF'
    WHEN 10 THEN 'DOC'
    WHEN 11 THEN 'SOB'
    WHEN 12 THEN 'DEC'
    WHEN 13 THEN 'ALU'
    WHEN 14 THEN 'FER'
    WHEN 15 THEN 'OBS'
    ELSE UPPER(SUBSTR(name, 1, 3))
END
WHERE internal_code_prefix = '';
