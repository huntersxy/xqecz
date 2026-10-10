export interface PlanetScreenPoint { x: number; y: number; diameter: number }
/** 信息卡跟随星球，靠近边缘时翻到另一侧；完全离开画面则隐藏。 */
export function universeSelectionPosition(point: PlanetScreenPoint, width: number, height: number, cardWidth: number, cardHeight: number) {
  const margin = 16, gap = 18, radius = point.diameter / 2
  const visible = point.x + radius > 0 && point.x - radius < width && point.y + radius > 0 && point.y - radius < height
  let x = point.x + radius + gap
  if (x + cardWidth > width - margin) x = point.x - radius - gap - cardWidth
  x = Math.max(margin, Math.min(width - cardWidth - margin, x))
  const y = Math.max(Math.min(124, height - cardHeight - margin), Math.min(height - cardHeight - margin, point.y - cardHeight / 2))
  return { x, y, visible }
}
