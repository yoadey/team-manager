import { z } from 'zod';
import { t } from '@/i18n';

export const newsFormSchema = z.object({
  id: z.string().optional(),
  title: z.string().trim().min(1, { message: t('news.fieldTitleError') }).max(255),
  body: z.string().trim().min(1, { message: t('news.fieldBodyError') }).max(10000),
  pinned: z.boolean().optional(),
});

export type NewsFormValues = z.infer<typeof newsFormSchema>;
