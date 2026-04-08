export function getSelectedPlayableModel(models, currentModel) {
  if (!Array.isArray(models) || models.length === 0) {
    return '';
  }

  if (models.includes(currentModel)) {
    return currentModel;
  }

  return models[0];
}
