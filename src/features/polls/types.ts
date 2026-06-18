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
