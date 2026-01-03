export interface Folder {
    id: number;
    name: string;
    path: string;
    directoryId: string;
}

export interface Child {
    id: string;
    title: string;
    isDir: boolean;
    coverArt?: string;
    path: string;
    artist?: string;
    album?: string;
    duration?: number;
    size?: number;
    track?: number;
    year?: number;
    genre?: string;
    bitRate?: number;
    suffix?: string;
}
export interface Directory {
    id: string;
    name: string;
    child: Child[];
}
