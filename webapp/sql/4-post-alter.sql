
ALTER TABLE `chairs` ADD COLUMN `is_occupied` TINYINT(1) NOT NULL DEFAULT 0 AFTER `is_active`;
ALTER TABLE `chairs` ADD INDEX (`is_active`, `is_occupied`);
