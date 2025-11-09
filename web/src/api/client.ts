// If not provided, use same-origin ('') which works with Vite dev proxy
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

type HttpMethod = 'GET' | 'POST' | 'DELETE' | 'PUT'

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

export interface PlanPercentsResponse {
  status: string
  data: {
    title: string
    date: string
    current_choice: number
    plans: string[] | null
    plan: number[] | null
    work: number[] | null
    learn: number[] | null
    rest: number[] | null
  }
}

export type PlanPercentGroup = 'plan' | 'work' | 'learn' | 'rest'

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

// Schedule types
export interface ScheduleTask {
  name: string
  role: 'work' | 'learn' | 'rest'
  time: number
  priority: number
  percents?: number[]
}

export interface DaySchedule {
  day: string
  total_time: number
  tasks: ScheduleTask[]
  plan_group: string[]
}

export interface WeeklySchedule {
  id?: string
  title: string
  created_at?: string
  updated_at?: string
  is_active: boolean
  monday: DaySchedule
  tuesday: DaySchedule
  wednesday: DaySchedule
  thursday: DaySchedule
  friday: DaySchedule
  saturday: DaySchedule
  sunday: DaySchedule
}

export interface ScheduleRequest {
  schedule: Omit<WeeklySchedule, 'id' | 'title' | 'created_at' | 'updated_at' | 'is_active'>
  set_active?: boolean
}

export interface RolloverTask {
  task_name: string
  role: string
  priority: number
  remaining_time: number
  source_day: string
  percent: number
}

export interface ActiveSchedule {
  day: string
  total_time: number
  tasks: ScheduleTask[]
  rollover_tasks: RolloverTask[]
  plan_group: string[]
}

export interface ScheduleResponse {
  status: string
  data?: WeeklySchedule
  message?: string
}

export interface ActiveScheduleResponse {
  status: string
  data: ActiveSchedule
}

export interface RolloverResponse {
  status: string
  data: {
    day: string
    rollover_tasks: RolloverTask[]
    count: number
  }
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
  getPlanPercents: () => request<PlanPercentsResponse>('GET', '/api/v1/manage/plan-percents'),
  changeLegacyPlanPercent: () => request<SuccessResponse>('GET', '/api/v1/task/plan-percent/change'),
  setProcents: (procents: number[], role_name?: string) =>
    request<SuccessResponse>('POST', '/api/v1/manage/procents', { procents, role_name }),
  removePlanPercent: (group: PlanPercentGroup, value: number) =>
    request<SuccessResponse>('DELETE', `/api/v1/manage/plan-percents/${group}/${value}`),

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

  // Schedule
  createSchedule: (payload: ScheduleRequest) => 
    request<{ status: string; data: { schedule_id: string; is_active: boolean }; message: string }>('POST', '/api/v1/schedule', payload),
  getActiveSchedule: () => 
    request<{ status: string; data: WeeklySchedule }>('GET', '/api/v1/schedule/active'),
  getSchedule: (id: string) => 
    request<{ status: string; data: WeeklySchedule }>('GET', `/api/v1/schedule/${id}`),
  updateSchedule: (id: string, schedule: Omit<WeeklySchedule, 'id' | 'title' | 'created_at' | 'updated_at' | 'is_active'>) => 
    request<SuccessResponse>('PUT', `/api/v1/schedule/${id}`, schedule),
  deleteSchedule: (id: string) => 
    request<SuccessResponse>('DELETE', `/api/v1/schedule/${id}`),
  activateSchedule: (id: string) => 
    request<SuccessResponse>('PUT', `/api/v1/schedule/${id}/activate`),
  getTodaySchedule: () => 
    request<ActiveScheduleResponse>('GET', '/api/v1/schedule/active/today'),
  getRolloverTasks: (day?: string) => 
    request<RolloverResponse>('GET', `/api/v1/schedule/active/rollover${day ? `?day=${day}` : ''}`),
  applySchedule: () => 
    request<SuccessResponse>('POST', '/api/v1/schedule/apply'),
}
