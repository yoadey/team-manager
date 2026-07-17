import { z } from 'zod';
import { t } from '@/i18n';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';

export const contribFormSchema = z.object({
  id: z.string(),
  label: z
    .string()
    .trim()
    .min(1, { message: t('finances.contribFieldLabelError') })
    .max(255),
  amount: z.string().superRefine((val, ctx) => {
    const res = validateMoneyAmount(val, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!res.ok) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: res.message,
      });
    }
  }),
});

export type ContribFormValues = z.infer<typeof contribFormSchema>;
