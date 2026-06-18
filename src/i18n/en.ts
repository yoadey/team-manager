import type { Messages } from './index';

// English skeleton. Kept structurally identical to the German reference so the
// type checker flags any drift. Translations can be refined later.
export const en: Messages = {
  common: {
    retry: 'Try again',
    cancel: 'Cancel',
    confirm: 'Confirm',
    save: 'Save',
    delete: 'Delete',
    loading: 'Loading…',
  },
  error: {
    generic: 'Something went wrong',
    action: 'Action failed',
    load: 'Could not load data',
    save: 'Saving failed',
    delete: 'Deleting failed',
    login: 'Sign-in failed',
    network: 'Could not connect to the service',
    unknown: 'Unknown error',
  },
  relTime: {
    now: 'just now',
    minutes: '{n} min ago',
    hours: '{n} h ago',
    day: '1 day ago',
    days: '{n} days ago',
  },
  eventType: {
    training: 'Training',
    auftritt: 'Performance / Tournament',
    event: 'Team event',
  },
  attendance: {
    yes: 'Attending',
    maybe: 'Maybe',
    no: 'Declined',
    pending: 'Open',
    not_nominated: 'Not nominated',
  },
};
