// If not provided, use same-origin ('') which works with Vite dev proxy
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

type HttpMethod = 'GET' | 'POST'

async function request<T>(method: HttpMethod, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  const data = text ? JSON.parse(text) : null
  if (!res.ok) {
    const msg = data?.message || `HTTP ${res.status}`
    throw new Error(msg)
  }
  return data as T
}

// Types (aligned with openapi.yml)
export interface TaskResult {
  name: string
  role: string
  time_duration: number
  time_done: number
  priority: number
}

export interface PlanPercentResponse {
  task_name: string
  percent: number
  time_left: number
}

export interface RestTimeResponse { rest_time: number }
export interface SuccessResponse { status: string; message?: string }

export interface TaskRecordRequest { task_name: string; time_done: number }
export interface RestRecordRequest { rest_time: number }
export interface CreateTaskRequest { task_name: string; role: string }
export interface TimerSetRequest { count: number }
export interface TimerResponse { time_duration: number }

// Records summary (today/yesterday/all)
export interface RecordsSummary {
  today: Record<string, number>
  yesterday: Record<string, number>
  all: Record<string, number>
}

// API wrappers
export const api = {
  // Statistics
  getStatsDoneToday: () => request<TaskResult[]>('GET', '/api/v1/stats/done/today'),
  getStatsTasksToday: () => request<TaskResult[]>('GET', '/api/v1/stats/tasks/today'),
  // Records summary
  getRecordsSummary: () => request<RecordsSummary>('GET', '/api/v1/records'),
  // Task list with planned vs done (today)
  getTaskList: () => request<TaskResult[]>('GET', '/api/v1/tasklist'),

  // Task plan percent
  getTaskPlanPercent: () => request<PlanPercentResponse>('GET', '/api/v1/task/plan/percent'),
  changeLegacyPlanPercent: () => request<SuccessResponse>('GET', '/api/v1/task/plan-percent/change'),
  setProcents: (procents: number[], role_name?: string) =>
    request<SuccessResponse>('POST', '/api/v1/manage/procents', { procents, role_name }),

  // Records
  addTaskRecord: (payload: TaskRecordRequest) => request<SuccessResponse>('POST', '/api/v1/taskrecord', payload),

  // Rest
  restGet: () => request<RestTimeResponse>('GET', '/api/v1/rest/get'),
  restAdd: (payload: RestRecordRequest) => request<SuccessResponse>('POST', '/api/v1/rest/add', payload),
  restSpend: (payload: RestRecordRequest) => request<SuccessResponse>('POST', '/api/v1/rest/spend', payload),

  // Manage
  createTask: (payload: CreateTaskRequest) => request<SuccessResponse>('POST', '/api/v1/manage/task/create', payload),

  // Timer
  timerGet: () => request<TimerResponse>('GET', '/api/v1/timer/get'),
  timerSet: (payload: TimerSetRequest) => request<SuccessResponse>('POST', '/api/v1/timer/set', payload),
}
