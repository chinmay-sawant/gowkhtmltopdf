package pdf

import "fmt"

// outputIntentDict builds the /OutputIntent dictionary for PDF/A-3 archival compliance.
func outputIntentDict(iccProfileRef objRef) string {
	return fmt.Sprintf(
		"<< /Type /OutputIntent /S /GTS_PDFA1 /OutputConditionIdentifier (sRGB IEC61966-2.1) "+
			"/Info (sRGB IEC61966-2.1) /DestOutputProfile %s >>",
		iccProfileRef.String(),
	)
}
