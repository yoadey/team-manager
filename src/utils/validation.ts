export interface ValidationResult<T = unknown> {
  ok: boolean;
  message?: string;
  value?: T;
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
const TIME_RE = /^\d{2}:\d{2}$/;
const MIN_REPEAT_WEEKS = 2;
const MAX_REPEAT_WEEKS = 26;

type MoneyOptions = {
  field?: string;
  positive?: boolean;
  allowZero?: boolean;
};

type EventForm = {
  title?: unknown;
  date?: unknown;
  meetT?: unknown;
  startT?: unknown;
  endT?: unknown;
  recurring?: unknown;
  repeatWeeks?: unknown;
};

type PollForm = {
  question?: unknown;
  opt0?: unknown;
  opt1?: unknown;
  opt2?: unknown;
  opt3?: unknown;
};

const text = (value: unknown) => String(value ?? '').trim();

const validDate = (value: string) => {
  if (!DATE_RE.test(value)) return false;
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
};

const minutes = (value: string) => {
  if (!TIME_RE.test(value)) return null;
  const [h, m] = value.split(':').map(Number);
  if (h < 0 || h > 23 || m < 0 || m > 59) return null;
  return h * 60 + m;
};

export function validateMoneyAmount(value: unknown, options: MoneyOptions = {}): ValidationResult<number> {
  const field = options.field || 'Betrag';
  const raw = text(value).replace(',', '.');
  if (!raw) return { ok: false, message: `${field} fehlt.` };
  const amount = Number(raw);
  if (!Number.isFinite(amount)) return { ok: false, message: `${field} muss eine gültige Zahl sein.` };
  const rounded = Math.round((amount + Number.EPSILON) * 100) / 100;
  if (options.positive && rounded <= 0) return { ok: false, message: `${field} muss größer als 0 € sein.` };
  if (!options.positive && options.allowZero !== false && rounded < 0) return { ok: false, message: `${field} darf nicht negativ sein.` };
  if (!options.positive && options.allowZero === false && rounded <= 0) return { ok: false, message: `${field} muss größer als 0 € sein.` };
  return { ok: true, value: rounded };
}

export function validateDateRange(from: unknown, to: unknown): ValidationResult<{ from: string; to: string }> {
  const start = text(from);
  const end = text(to);
  if (!start) return { ok: false, message: 'Startdatum fehlt.' };
  if (!end) return { ok: false, message: 'Enddatum fehlt.' };
  if (!validDate(start)) return { ok: false, message: 'Startdatum ist ungültig.' };
  if (!validDate(end)) return { ok: false, message: 'Enddatum ist ungültig.' };
  if (end < start) return { ok: false, message: 'Enddatum darf nicht vor dem Startdatum liegen.' };
  return { ok: true, value: { from: start, to: end } };
}

export function validateEventForm(form: EventForm, mode: 'create' | 'edit' = 'create'): ValidationResult<{ repeatWeeks: number }> {
  if (!text(form.title)) return { ok: false, message: 'Titel des Termins fehlt.' };
  const date = text(form.date);
  if (!date) return { ok: false, message: 'Datum des Termins fehlt.' };
  if (!validDate(date)) return { ok: false, message: 'Datum des Termins ist ungültig.' };

  const start = text(form.startT);
  const end = text(form.endT);
  const meet = text(form.meetT);
  const startMin = start ? minutes(start) : null;
  const endMin = end ? minutes(end) : null;
  const meetMin = meet ? minutes(meet) : null;

  if (start && startMin == null) return { ok: false, message: 'Beginn des Termins ist ungültig.' };
  if (end && endMin == null) return { ok: false, message: 'Ende des Termins ist ungültig.' };
  if (meet && meetMin == null) return { ok: false, message: 'Treffzeit ist ungültig.' };
  if (startMin != null && endMin != null && endMin <= startMin) return { ok: false, message: 'Ende muss nach dem Beginn liegen.' };
  if (meetMin != null && startMin != null && meetMin > startMin) return { ok: false, message: 'Treffzeit darf nicht nach dem Beginn liegen.' };

  const repeatWeeks = Number(form.repeatWeeks);
  if (mode === 'create' && form.recurring) {
    if (!Number.isInteger(repeatWeeks)) return { ok: false, message: 'Serienwochen müssen eine ganze Zahl sein.' };
    if (repeatWeeks < MIN_REPEAT_WEEKS || repeatWeeks > MAX_REPEAT_WEEKS) return { ok: false, message: `Serienwochen müssen zwischen ${MIN_REPEAT_WEEKS} und ${MAX_REPEAT_WEEKS} liegen.` };
  }
  return { ok: true, value: { repeatWeeks: Number.isFinite(repeatWeeks) ? repeatWeeks : MIN_REPEAT_WEEKS } };
}

export function validatePollForm(form: PollForm): ValidationResult<{ question: string; options: string[] }> {
  const question = text(form.question);
  if (!question) return { ok: false, message: 'Frage der Umfrage fehlt.' };
  const options = [form.opt0, form.opt1, form.opt2, form.opt3].map(text).filter(Boolean);
  if (options.length < 2) return { ok: false, message: 'Mindestens zwei Antwortoptionen angeben.' };
  if (new Set(options.map((o) => o.toLocaleLowerCase('de'))).size !== options.length) return { ok: false, message: 'Antwortoptionen dürfen nicht doppelt sein.' };
  return { ok: true, value: { question, options } };
}

export function validateRequiredText(value: unknown, message: string): ValidationResult<string> {
  const cleaned = text(value);
  return cleaned ? { ok: true, value: cleaned } : { ok: false, message };
}
