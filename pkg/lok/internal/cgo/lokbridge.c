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
    pOffice->pClass->destroy(pOffice);
}

char* lok_bridge_get_error(LoKit* pOffice) {
    return pOffice->pClass->getError(pOffice);
}

void lok_bridge_free_error(LoKit* pOffice, char* pErr) {
    if (LIBREOFFICEKIT_HAS(pOffice, freeError)) {
        pOffice->pClass->freeError(pErr);
    } else {
        free(pErr);
    }
}

char* lok_bridge_get_version_info(LoKit* pOffice) {
    return pOffice->pClass->getVersionInfo(pOffice);
}

char* lok_bridge_get_filter_types(LoKit* pOffice) {
    return pOffice->pClass->getFilterTypes(pOffice);
}

// trimMemory was added in LibreOffice 7.6. The struct member may not exist
// in older headers, so we check the vtable size at runtime rather than using
// LIBREOFFICEKIT_HAS (which requires compile-time struct member existence).
int lok_bridge_has_trim_memory(LoKit* pOffice) {
#ifdef LOK_HAS_TRIM_MEMORY
    return LIBREOFFICEKIT_HAS(pOffice, trimMemory);
#else
    (void)pOffice;
    return 0;
#endif
}

void lok_bridge_trim_memory(LoKit* pOffice, int nTarget) {
#ifdef LOK_HAS_TRIM_MEMORY
    if (LIBREOFFICEKIT_HAS(pOffice, trimMemory)) {
        pOffice->pClass->trimMemory(pOffice, nTarget);
    }
#else
    (void)pOffice;
    (void)nTarget;
#endif
}

LoKitDocument* lok_bridge_document_load(LoKit* pOffice, const char* pURL) {
    return pOffice->pClass->documentLoad(pOffice, pURL);
}

LoKitDocument* lok_bridge_document_load_with_options(LoKit* pOffice,
                                                     const char* pURL,
                                                     const char* pOptions) {
    return pOffice->pClass->documentLoadWithOptions(pOffice, pURL, pOptions);
}

void lok_bridge_document_destroy(LoKitDocument* pDoc) {
    pDoc->pClass->destroy(pDoc);
}

// LOK saveAs returns 0 on failure, unlike typical C convention.
int lok_bridge_document_save_as(LoKitDocument* pDoc, const char* pURL,
                                const char* pFormat,
                                const char* pFilterOptions) {
    return pDoc->pClass->saveAs(pDoc, pURL, pFormat, pFilterOptions);
}

int lok_bridge_document_get_type(LoKitDocument* pDoc) {
    return pDoc->pClass->getDocumentType(pDoc);
}

void lok_bridge_document_post_uno_command(LoKitDocument* pDoc,
                                          const char* pCommand,
                                          const char* pArguments,
                                          int bNotifyWhenFinished) {
    pDoc->pClass->postUnoCommand(pDoc, pCommand, pArguments,
                                 (bool)bNotifyWhenFinished);
}
