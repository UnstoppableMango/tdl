package check;

import java.io.File;
import org.contextmapper.dsl.standalone.ContextMapperStandaloneSetup;
import org.contextmapper.dsl.standalone.StandaloneContextMapperAPI;
import org.contextmapper.dsl.cml.CMLResource;
import org.eclipse.emf.ecore.resource.Resource;
import org.eclipse.xtext.validation.CheckMode;
import org.eclipse.xtext.validation.IResourceValidator;
import org.eclipse.xtext.validation.Issue;
import org.eclipse.xtext.resource.XtextResource;

public class Main {
  public static void main(String[] args) {
    StandaloneContextMapperAPI api = ContextMapperStandaloneSetup.getStandaloneAPI();
    int bad = 0;
    for (String a : args) {
      CMLResource r = api.loadCML(new File(a));
      XtextResource res = r.getXtextResource();
      IResourceValidator v = res.getResourceServiceProvider().getResourceValidator();
      for (Issue i : v.validate(res, CheckMode.ALL, null)) {
        System.out.println(a + ":" + i.getLineNumber() + ": " + i.getSeverity() + ": " + i.getMessage());
        if (i.getSeverity() == org.eclipse.xtext.diagnostics.Severity.ERROR) bad++;
      }
    }
    System.exit(bad == 0 ? 0 : 1);
  }
}
