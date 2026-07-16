import { z } from 'zod';
import { t } from '@/i18n';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount } from '@/utils/validation';

export const txFormSchema = z.object({
  id: z.string().optional(),
  type: z.enum(['income', 'expense']),
  title: z.string().trim().min(1, { message: t('finances.txFieldTitleError') }).max(255),
  amount: z.string().superRefine((val, ctx) => {
    const res = validateMoneyAmount(val, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!res.ok) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: res.message,
      });
    }
  }),
  category: z.string().trim().max(255).optional().or(z.literal('')),
});

export type TxFormValues = z.infer<typeof txFormSchema>;
