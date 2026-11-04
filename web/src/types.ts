export type Container = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: string;
  labels?: Record<string, string>;
};

export type LogLine = {
  id: string;
  raw: string;
  ts?: string;
  body: string;
};
