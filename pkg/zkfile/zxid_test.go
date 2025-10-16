package zkfile

import (
	"testing"
)

func TestFileType_String(t *testing.T) {
	tests := []struct {
		name string
		ft   FileType
		want string
	}{
		{
			name: "snapshot",
			ft:   FileTypeSnapshot,
			want: "snapshot",
		},
		{
			name: "txnlog",
			ft:   FileTypeTxnLog,
			want: "txnlog",
		},
		{
			name: "unknown",
			ft:   FileTypeUnknown,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ft.String(); got != tt.want {
				t.Errorf("FileType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZXID_String(t *testing.T) {
	tests := []struct {
		name string
		zxid ZXID
		want string
	}{
		{
			name: "zero",
			zxid: ZXID(0),
			want: "0x0",
		},
		{
			name: "small value",
			zxid: ZXID(0x100),
			want: "0x100",
		},
		{
			name: "large value",
			zxid: ZXID(0x100000000),
			want: "0x100000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.zxid.String(); got != tt.want {
				t.Errorf("ZXID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZXID_Hex(t *testing.T) {
	tests := []struct {
		name string
		zxid ZXID
		want string
	}{
		{
			name: "zero",
			zxid: ZXID(0),
			want: "0",
		},
		{
			name: "hex value",
			zxid: ZXID(0x100000000),
			want: "100000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.zxid.Hex(); got != tt.want {
				t.Errorf("ZXID.Hex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZXID_Compare(t *testing.T) {
	tests := []struct {
		name  string
		z     ZXID
		other ZXID
		want  int
	}{
		{
			name:  "equal",
			z:     ZXID(100),
			other: ZXID(100),
			want:  0,
		},
		{
			name:  "less than",
			z:     ZXID(100),
			other: ZXID(200),
			want:  -1,
		},
		{
			name:  "greater than",
			z:     ZXID(200),
			other: ZXID(100),
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.z.Compare(tt.other); got != tt.want {
				t.Errorf("ZXID.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseZxidFromFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     ZXID
		wantErr  bool
	}{
		{
			name:     "snapshot file",
			filename: "snapshot.100000000",
			want:     ZXID(0x100000000),
			wantErr:  false,
		},
		{
			name:     "log file",
			filename: "log.100000000",
			want:     ZXID(0x100000000),
			wantErr:  false,
		},
		{
			name:     "full path",
			filename: "/path/to/snapshot.200000000",
			want:     ZXID(0x200000000),
			wantErr:  false,
		},
		{
			name:     "invalid format",
			filename: "invalid",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "invalid zxid",
			filename: "snapshot.xyz",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZxidFromFileName(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseZxidFromFileName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseZxidFromFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineFileType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     FileType
	}{
		{
			name:     "snapshot file",
			filename: "snapshot.100000000",
			want:     FileTypeSnapshot,
		},
		{
			name:     "log file",
			filename: "log.100000000",
			want:     FileTypeTxnLog,
		},
		{
			name:     "unknown file",
			filename: "data.txt",
			want:     FileTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineFileType(tt.filename); got != tt.want {
				t.Errorf("DetermineFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxZXID(t *testing.T) {
	tests := []struct {
		name string
		a    ZXID
		b    ZXID
		want ZXID
	}{
		{
			name: "a greater",
			a:    ZXID(200),
			b:    ZXID(100),
			want: ZXID(200),
		},
		{
			name: "b greater",
			a:    ZXID(100),
			b:    ZXID(200),
			want: ZXID(200),
		},
		{
			name: "equal",
			a:    ZXID(100),
			b:    ZXID(100),
			want: ZXID(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxZXID(tt.a, tt.b); got != tt.want {
				t.Errorf("MaxZXID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinZXID(t *testing.T) {
	tests := []struct {
		name string
		a    ZXID
		b    ZXID
		want ZXID
	}{
		{
			name: "a smaller",
			a:    ZXID(100),
			b:    ZXID(200),
			want: ZXID(100),
		},
		{
			name: "b smaller",
			a:    ZXID(200),
			b:    ZXID(100),
			want: ZXID(100),
		},
		{
			name: "equal",
			a:    ZXID(100),
			b:    ZXID(100),
			want: ZXID(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinZXID(tt.a, tt.b); got != tt.want {
				t.Errorf("MinZXID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatZxidFileName(t *testing.T) {
	tests := []struct {
		name     string
		fileType FileType
		zxid     ZXID
		want     string
	}{
		{
			name:     "snapshot",
			fileType: FileTypeSnapshot,
			zxid:     ZXID(0x100000000),
			want:     "snapshot.100000000",
		},
		{
			name:     "txnlog",
			fileType: FileTypeTxnLog,
			zxid:     ZXID(0x200000000),
			want:     "log.200000000",
		},
		{
			name:     "unknown",
			fileType: FileTypeUnknown,
			zxid:     ZXID(0x100000000),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatZxidFileName(tt.fileType, tt.zxid); got != tt.want {
				t.Errorf("FormatZxidFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}
