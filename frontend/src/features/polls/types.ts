export interface PollOption {
  id: string;
  text: string;
  count: number;
  pct: number;
  voters: { name: string; color: string; photo: string | null }[];
}

export interface Poll {
  id: string;
  question: string;
  multiple: boolean;
  anonymous: boolean;
  createdAt: string;
  totalVotes: number;
  myVote: string[] | null;
  options: PollOption[];
}

/** Editing buffer shape for the poll-creation sheet. */
export interface PollFormValues extends Record<string, unknown> {
  question: string;
  opt0: string;
  opt1: string;
  opt2: string;
  opt3: string;
  multiple: boolean;
  anonymous: boolean;
}
