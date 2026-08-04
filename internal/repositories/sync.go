package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"buffetflow/internal/models"
)

func (s *Store) RegisterSyncDevice(ctx context.Context, deviceID, deviceName string, userID int64) error {
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_devices(device_id,user_id,device_name,last_seen_at,created_at) VALUES(?,?,?,?,?) ON CONFLICT(device_id) DO UPDATE SET user_id=excluded.user_id,device_name=excluded.device_name,last_seen_at=excluded.last_seen_at,revoked_at=NULL`, deviceID, userID, deviceName, now, now)
	return err
}
func (s *Store) ExistingSyncOperation(ctx context.Context, clientID string) (models.SyncOperationResult, bool, error) {
	var resultJSON string
	err := s.db.QueryRowContext(ctx, `SELECT result_json FROM sync_operations WHERE client_operation_id=?`, clientID).Scan(&resultJSON)
	if err == sql.ErrNoRows {
		return models.SyncOperationResult{}, false, nil
	}
	if err != nil {
		return models.SyncOperationResult{}, false, err
	}
	var result models.SyncOperationResult
	err = json.Unmarshal([]byte(resultJSON), &result)
	return result, true, err
}
func (s *Store) RecordSyncOperation(ctx context.Context, request models.SyncOperationRequest, userID int64, result models.SyncOperationResult) error {
	resultJSON, _ := json.Marshal(result)
	status := result.Status
	if status == "synced" {
		status = "applied"
	}
	var snapshot any
	if result.ServerSnapshot != nil {
		encoded, _ := json.Marshal(result.ServerSnapshot)
		snapshot = string(encoded)
	}
	now := nowString()
	if request.LocalDate == "" {
		request.LocalDate = time.Now().UTC().Format(time.RFC3339)
	}
	applied := any(nil)
	if status == "applied" {
		applied = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_operations(client_operation_id,device_id,user_id,operation_type,entity_type,entity_id,base_version,payload_json,status,result_json,server_snapshot_json,last_error,submitted_at,applied_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(client_operation_id) DO NOTHING`, request.ClientOperationID, request.DeviceID, userID, request.OperationType, request.EntityType, nullablePositiveID(request.EntityID), request.BaseVersion, string(request.Payload), status, string(resultJSON), snapshot, result.Error, request.LocalDate, applied, now, now)
	return err
}
func (s *Store) MarkDeviceSynced(ctx context.Context, deviceID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE sync_devices SET last_sync_at=?,last_seen_at=? WHERE device_id=?`, now, now, deviceID)
	return err
}
func nullablePositiveID(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
