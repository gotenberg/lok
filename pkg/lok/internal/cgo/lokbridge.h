#ifndef LOK_BRIDGE_H
#define LOK_BRIDGE_H

#define LOK_USE_UNSTABLE_API

#include <LibreOfficeKit/LibreOfficeKit.h>
#include <LibreOfficeKit/LibreOfficeKitInit.h>
#include <stdlib.h>

typedef LibreOfficeKit LoKit;
typedef LibreOfficeKitDocument LoKitDocument;

// PasswordState carries the password for the document being loaded to the
// LibreOfficeKit document-password callback.
typedef struct {
    LoKit* pOffice;
    char* pPassword;
    int nRequests;
} PasswordState;

// Office lifecycle.
LoKit* lok_bridge_init_2(const char* installPath, const char* userProfilePath);
void lok_bridge_destroy(LoKit* pOffice);

// Document passwords.
PasswordState* lok_bridge_enable_document_password(LoKit* pOffice);
void lok_bridge_free_password_state(PasswordState* pState);

// Error handling.
char* lok_bridge_get_error(LoKit* pOffice);
void lok_bridge_free_error(LoKit* pOffice, char* pErr);

// Info.
char* lok_bridge_get_version_info(LoKit* pOffice);

// Scripting.
int lok_bridge_run_macro(LoKit* pOffice, const char* pURL);

// Memory.
int lok_bridge_has_trim_memory(LoKit* pOffice);
void lok_bridge_trim_memory(LoKit* pOffice, int nTarget);

// Document loading.
LoKitDocument* lok_bridge_document_load_with_password(
    LoKit* pOffice, PasswordState* pState, const char* pURL,
    const char* pOptions, const char* pPassword);

// Document operations.
void lok_bridge_document_destroy(LoKitDocument* pDoc);
int lok_bridge_document_save_as(
    LoKitDocument* pDoc, const char* pURL,
    const char* pFormat, const char* pFilterOptions);
int lok_bridge_document_get_type(LoKitDocument* pDoc);

#endif // LOK_BRIDGE_H
