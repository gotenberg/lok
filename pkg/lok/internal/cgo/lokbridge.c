#include "lokbridge.h"

#include <stdbool.h>

LoKit* lok_bridge_init(const char* installPath) {
    return lok_init(installPath);
}

LoKit* lok_bridge_init_2(const char* installPath,
                         const char* userProfilePath) {
    return lok_init_2(installPath, userProfilePath);
}

void lok_bridge_destroy(LoKit* pOffice) {
    if (!pOffice) return;
    pOffice->pClass->destroy(pOffice);
}

char* lok_bridge_get_error(LoKit* pOffice) {
    if (!pOffice) return NULL;
    return pOffice->pClass->getError(pOffice);
}

void lok_bridge_free_error(LoKit* pOffice, char* pErr) {
    if (!pOffice || !pErr) return;
    if (LIBREOFFICEKIT_HAS(pOffice, freeError)) {
        pOffice->pClass->freeError(pErr);
    } else {
        free(pErr);
    }
}

char* lok_bridge_get_version_info(LoKit* pOffice) {
    if (!pOffice) return NULL;
    return pOffice->pClass->getVersionInfo(pOffice);
}

char* lok_bridge_get_filter_types(LoKit* pOffice) {
    if (!pOffice) return NULL;
    return pOffice->pClass->getFilterTypes(pOffice);
}

// LOK runMacro returns 0 on failure, unlike typical C convention.
int lok_bridge_run_macro(LoKit* pOffice, const char* pURL) {
    if (!pOffice) return 0;
    return pOffice->pClass->runMacro(pOffice, pURL);
}

// trimMemory was added in LibreOffice 7.6. Older LibreOfficeKit headers do not
// declare the struct member, so referencing it (even through LIBREOFFICEKIT_HAS,
// which uses offsetof) fails to compile. Build with -DLOK_HAS_TRIM_MEMORY (via
// CGO_CFLAGS) when compiling against LibreOffice 7.6+ headers to enable the
// feature; otherwise these functions degrade to no-ops. When the member is
// present, LIBREOFFICEKIT_HAS still guards against an older runtime.
int lok_bridge_has_trim_memory(LoKit* pOffice) {
#ifdef LOK_HAS_TRIM_MEMORY
    if (!pOffice) return 0;
    return LIBREOFFICEKIT_HAS(pOffice, trimMemory);
#else
    (void)pOffice;
    return 0;
#endif
}

void lok_bridge_trim_memory(LoKit* pOffice, int nTarget) {
#ifdef LOK_HAS_TRIM_MEMORY
    if (pOffice && LIBREOFFICEKIT_HAS(pOffice, trimMemory)) {
        pOffice->pClass->trimMemory(pOffice, nTarget);
    }
#else
    (void)pOffice;
    (void)nTarget;
#endif
}

LoKitDocument* lok_bridge_document_load(LoKit* pOffice, const char* pURL) {
    if (!pOffice) return NULL;
    return pOffice->pClass->documentLoad(pOffice, pURL);
}

LoKitDocument* lok_bridge_document_load_with_options(LoKit* pOffice,
                                                     const char* pURL,
                                                     const char* pOptions) {
    if (!pOffice) return NULL;
    return pOffice->pClass->documentLoadWithOptions(pOffice, pURL, pOptions);
}

void lok_bridge_document_destroy(LoKitDocument* pDoc) {
    if (!pDoc) return;
    pDoc->pClass->destroy(pDoc);
}

// LOK saveAs returns 0 on failure, unlike typical C convention.
int lok_bridge_document_save_as(LoKitDocument* pDoc, const char* pURL,
                                const char* pFormat,
                                const char* pFilterOptions) {
    if (!pDoc) return 0;
    return pDoc->pClass->saveAs(pDoc, pURL, pFormat, pFilterOptions);
}

int lok_bridge_document_get_type(LoKitDocument* pDoc) {
    if (!pDoc) return -1;
    return pDoc->pClass->getDocumentType(pDoc);
}

void lok_bridge_document_post_uno_command(LoKitDocument* pDoc,
                                          const char* pCommand,
                                          const char* pArguments,
                                          int bNotifyWhenFinished) {
    if (!pDoc) return;
    pDoc->pClass->postUnoCommand(pDoc, pCommand, pArguments,
                                 (bool)bNotifyWhenFinished);
}
