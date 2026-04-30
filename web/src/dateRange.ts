export type DatePreset =
  | 'today'
  | 'yesterday'
  | 'this_week'
  | 'last_week'
  | 'this_month'
  | 'last_month'
  | 'custom'

export type DateRangeValue =
  | { preset: Exclude<DatePreset, 'custom'> }
  | { preset: 'custom'; from: string; to: string }

export function resolveDateRange(value: DateRangeValue): { from: Date; to: Date } {
  const now = new Date()

  switch (value.preset) {
    case 'today':
      return dayRange(now)
    case 'yesterday': {
      const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
      return dayRange(yesterday)
    }
    case 'this_week': {
      return { from: startOfWeek(now), to: endOfDay(now) }
    }
    case 'last_week': {
      const thisWeekStart = startOfWeek(now)
      const lastWeekStart = new Date(
        thisWeekStart.getFullYear(),
        thisWeekStart.getMonth(),
        thisWeekStart.getDate() - 7,
      )
      const lastWeekEnd = new Date(
        lastWeekStart.getFullYear(),
        lastWeekStart.getMonth(),
        lastWeekStart.getDate() + 6,
      )
      return { from: startOfDay(lastWeekStart), to: endOfDay(lastWeekEnd) }
    }
    case 'this_month':
      return { from: new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0), to: endOfDay(now) }
    case 'last_month': {
      const start = new Date(now.getFullYear(), now.getMonth() - 1, 1, 0, 0, 0, 0)
      const end = new Date(now.getFullYear(), now.getMonth(), 0)
      return { from: start, to: endOfDay(end) }
    }
    case 'custom':
      return { from: parseDateInputStart(value.from), to: parseDateInputEnd(value.to) }
  }
}

function dayRange(date: Date) {
  return { from: startOfDay(date), to: endOfDay(date) }
}

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0)
}

function endOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59, 999)
}

function startOfWeek(date: Date) {
  const day = date.getDay()
  const daysSinceMonday = (day + 6) % 7
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() - daysSinceMonday, 0, 0, 0, 0)
}

function parseDateInputStart(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day, 0, 0, 0, 0)
}

function parseDateInputEnd(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day, 23, 59, 59, 999)
}
