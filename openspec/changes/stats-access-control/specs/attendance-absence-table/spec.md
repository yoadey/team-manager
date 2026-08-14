## REMOVED Requirements

### Requirement: A per-member, per-event absence table is available alongside the quota view
Reason: the table listed every absence with its event and date but surfaced no information beyond what the quota and matrix views already show in a more useful, aggregated form; it was not worth maintaining.
Migration: the `GET /teams/{teamId}/stats/absences` endpoint, its `AttendanceAbsenceTable`/`AttendanceAbsenceRow` schemas, the backend service/repository code that served it, and the frontend "Fehlzeiten" tab are removed entirely. No data is lost — the underlying absence records remain visible via the events/absences views.
