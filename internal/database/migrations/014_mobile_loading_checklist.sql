ALTER TABLE checklist_items ADD COLUMN loading_decision TEXT CHECK(loading_decision IN ('complete','missing'));
ALTER TABLE checklist_items ADD COLUMN loading_missing_quantity REAL NOT NULL DEFAULT 0 CHECK(loading_missing_quantity >= 0);

UPDATE checklist_items
SET loading_decision = CASE
        WHEN loaded_quantity >= required_quantity THEN 'complete'
        ELSE 'missing'
    END,
    loading_missing_quantity = MAX(0, required_quantity - loaded_quantity)
WHERE status IN ('loaded','at_event','returned','damaged','lost')
  AND required_quantity > 0;
