export const isUserDeleted = (record) =>
  record?.DeletedAt != null || record?.deleted_at != null;
