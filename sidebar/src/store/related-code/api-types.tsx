export interface Push {
  editor: string
  editor_install_path: string
  location: Location
  relative_filename: string
  filename: string
  relative_path: string
  project_tag: string
}

export interface FetchRequest {
  location: Location
  num_blocks?: number
  num_files: number
  num_keywords?: number
  offset?: number
  editor: string
}

export interface FetchResponse {
  project_root: string
  related_files: RelatedFile[]
  filename: string
  relative_path: string
}

export interface OpenFileRequest {
  // Absolute path of the editor's binary
  path: string,
  // Absolute path of the file to open
  filename: string,
  line: number,

  // telemetry fields
  block_rank?: number,
  file_rank: number,
}

export interface RelatedFile {
  file: File
  filename: string
  relative_path: string
}

export interface File {
  blocks: Block[]
  absolute_path: string
  keywords: Keyword[]
  score: number
}

export interface Block {
  content: string
  firstline: number
  keywords: Keyword[]
  lastline: number
  score: number
}

export interface Keyword {
  keyword: string
}

export interface Location {
  filename: string
  line: number
}