#ifndef LOK_BRIDGE_H
#define LOK_BRIDGE_H

#define LOK_USE_UNSTABLE_API

#include <LibreOfficeKit/LibreOfficeKit.h>
#include <LibreOfficeKit/LibreOfficeKitInit.h>
#include <stdlib.h>

typedef LibreOfficeKit LoKit;
typedef LibreOfficeKitDocument LoKitDocument;

// Office lifecycle.
LoKit* lok_bridge_init(const char* installPath);
LoKit* lok_bridge_init_2(const char* installPath, const char* userProfilePath);
void lok_bridge_destroy(LoKit* pOffice);

// Error handling.
char* lok_bridge_get_error(LoKit* pOffice);
void lok_bridge_free_error(LoKit* pOffice, char* pErr);

// Info.
char* lok_bridge_get_version_info(LoKit* pOffice);
char* lok_bridge_get_filter_types(LoKit* pOffice);

// Memory.
int lok_bridge_has_trim_memory(LoKit* pOffice);
void lok_bridge_trim_memory(LoKit* pOffice, int nTarget);

// Document loading.
LoKitDocument* lok_bridge_document_load(LoKit* pOffice, const char* pURL);
LoKitDocument* lok_bridge_document_load_with_options(
    LoKit* pOffice, const char* pURL, const char* pOptions);

// Document operations.
void lok_bridge_document_destroy(LoKitDocument* pDoc);
int lok_bridge_document_save_as(
    LoKitDocument* pDoc, const char* pURL,
    const char* pFormat, const char* pFilterOptions);
int lok_bridge_document_get_type(LoKitDocument* pDoc);
void lok_bridge_document_post_uno_command(
    LoKitDocument* pDoc, const char* pCommand,
    const char* pArguments, int bNotifyWhenFinished);

#endif // LOK_BRIDGE_H
