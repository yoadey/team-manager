import { z } from 'zod';
import { t } from '@/i18n';

export const pollFormSchema = z
  .object({
    question: z
      .string()
      .trim()
      .min(1, { message: t('polls.fieldQuestionError') })
      .max(1000),
    opt0: z.string().trim().max(500).optional().or(z.literal('')),
    opt1: z.string().trim().max(500).optional().or(z.literal('')),
    opt2: z.string().trim().max(500).optional().or(z.literal('')),
    opt3: z.string().trim().max(500).optional().or(z.literal('')),
    multiple: z.boolean(),
    anonymous: z.boolean(),
  })
  .superRefine((data, ctx) => {
    const opts = [data.opt0, data.opt1, data.opt2, data.opt3].map((o) => String(o ?? '').trim()).filter(Boolean);

    if (opts.length < 2) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['options'],
        message: t('polls.optionsError'),
      });
    } else {
      const lowercased = opts.map((o) => o.toLocaleLowerCase('de'));
      const hasDuplicates = new Set(lowercased).size !== opts.length;
      if (hasDuplicates) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['options'],
          message: t('validation.pollOptionsDuplicate'),
        });
      }
    }
  });

export type PollFormValues = z.infer<typeof pollFormSchema>;
